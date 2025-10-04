package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jpoz/taco/internal/config"
	"github.com/jpoz/taco/internal/recorder"
)

type Proxy struct {
	cfg      *config.Config
	client   *http.Client
	recorder *recorder.Recorder
}

func New(cfg *config.Config, rec *recorder.Recorder) *Proxy {
	return &Proxy{
		cfg:      cfg,
		recorder: rec,
		client: &http.Client{
			Timeout: 300 * time.Second, // Longer timeout for streaming
		},
	}
}

func (p *Proxy) identifyProvider(path string) string {
	// Claude endpoints start with /v1/messages or /v1/complete
	if strings.HasPrefix(path, "/v1/messages") || strings.HasPrefix(path, "/v1/complete") {
		return "claude"
	}
	// OpenAI endpoints
	if strings.HasPrefix(path, "/v1/chat/completions") ||
		strings.HasPrefix(path, "/v1/completions") ||
		strings.HasPrefix(path, "/v1/embeddings") ||
		strings.HasPrefix(path, "/v1/models") {
		return "openai"
	}
	return ""
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	provider := p.identifyProvider(r.URL.Path)
	if provider == "" {
		http.Error(w, "unknown API endpoint", http.StatusNotFound)
		return
	}

	providerCfg, ok := p.cfg.Providers[provider]
	if !ok {
		http.Error(w, fmt.Sprintf("provider %s not configured", provider), http.StatusInternalServerError)
		return
	}

	// Read and capture request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Create recording
	rec := recorder.NewRecording(provider, r.Method, r.URL.Path, startTime)
	rec.Request.Headers = r.Header.Clone()
	if len(bodyBytes) > 0 {
		// Try to parse as JSON, otherwise store as string
		var jsonBody interface{}
		if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
			rec.Request.Body = jsonBody
		} else {
			rec.Request.Body = string(bodyBytes)
		}
	}

	// Create upstream request
	upstreamURL := providerCfg.UpstreamURL + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make upstream request
	resp, err := p.client.Do(req)
	if err != nil {
		slog.Error("upstream request failed", "error", err, "provider", provider, "path", r.URL.Path)
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	rec.Response.Status = resp.StatusCode
	rec.Response.Headers = resp.Header.Clone()

	// Check if streaming
	isStreaming := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") ||
		strings.Contains(resp.Header.Get("Content-Type"), "stream")
	rec.Response.Streaming = isStreaming

	w.WriteHeader(resp.StatusCode)

	if isStreaming {
		p.handleStreaming(w, resp.Body, &rec)
	} else {
		p.handleRegular(w, resp.Body, &rec)
	}

	rec.Timing.CompletedAt = time.Now()
	rec.Timing.DurationMs = rec.Timing.CompletedAt.Sub(rec.Timing.StartedAt).Milliseconds()

	// Log completion
	logLevel := slog.LevelInfo
	if rec.Response.Status >= 400 {
		logLevel = slog.LevelError
	} else if rec.Response.Status >= 300 {
		logLevel = slog.LevelWarn
	}

	slog.Log(r.Context(), logLevel, "request completed",
		"id", rec.ID[:8],
		"provider", rec.Provider,
		"status", rec.Response.Status,
		"duration_ms", rec.Timing.DurationMs,
		"path", rec.Request.Path)

	// Record asynchronously
	p.recorder.Record(rec)
}

func (p *Proxy) handleRegular(w http.ResponseWriter, body io.Reader, rec *recorder.Recording) {
	var buf bytes.Buffer
	tee := io.TeeReader(body, &buf)

	if _, err := io.Copy(w, tee); err != nil {
		slog.Error("failed to copy response", "error", err)
		return
	}

	if buf.Len() > 0 {
		// Try to parse as JSON, otherwise store as string
		var jsonBody interface{}
		if err := json.Unmarshal(buf.Bytes(), &jsonBody); err == nil {
			rec.Response.Body = jsonBody
		} else {
			rec.Response.Body = string(buf.Bytes())
		}
	}
}

func (p *Proxy) handleStreaming(w http.ResponseWriter, body io.Reader, rec *recorder.Recording) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("response writer does not support flushing")
		io.Copy(w, body)
		return
	}

	var accumulated bytes.Buffer
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // Support large chunks

	for scanner.Scan() {
		line := scanner.Bytes()
		accumulated.Write(line)
		accumulated.WriteByte('\n')

		// Write to client
		if _, err := w.Write(line); err != nil {
			slog.Error("failed to write streaming chunk", "error", err)
			break
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			slog.Error("failed to write newline", "error", err)
			break
		}
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil {
		slog.Error("error reading stream", "error", err)
	}

	if accumulated.Len() > 0 {
		// Store streaming responses as string (they contain SSE format)
		rec.Response.Body = accumulated.String()
	}
}
