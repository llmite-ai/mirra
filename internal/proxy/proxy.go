package proxy

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/llmite-ai/mirra/internal/config"
	"github.com/llmite-ai/mirra/internal/recorder"
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
	// Gemini endpoints - check before OpenAI to avoid /v1/models conflict
	if isGeminiPath(path) {
		return "gemini"
	}
	// OpenAI endpoints
	if strings.HasPrefix(path, "/v1/chat/completions") ||
		strings.HasPrefix(path, "/v1/completions") ||
		strings.HasPrefix(path, "/v1/embeddings") ||
		strings.HasPrefix(path, "/v1/models") ||
		strings.HasPrefix(path, "/v1/responses") {
		return "openai"
	}
	return ""
}

// isGeminiPath checks if the path matches any Gemini API endpoint pattern.
// Supports v1, v1beta, and v1alpha API versions.
func isGeminiPath(path string) bool {
	// Check for API version prefixes
	hasVersion := strings.HasPrefix(path, "/v1/") ||
		strings.HasPrefix(path, "/v1beta/") ||
		strings.HasPrefix(path, "/v1alpha/")

	if !hasVersion {
		// Special case: file upload uses /upload/v1* prefix
		if strings.HasPrefix(path, "/upload/v1/") ||
			strings.HasPrefix(path, "/upload/v1beta/") ||
			strings.HasPrefix(path, "/upload/v1alpha/") {
			return strings.Contains(path, "/files")
		}
		return false
	}

	// Model operations: /v1*/models/*
	// Includes: generateContent, streamGenerateContent, embedContent, countTokens, etc.
	if strings.Contains(path, "/models/") || strings.Contains(path, "/models:") {
		return true
	}

	// File operations: /v1*/files, /v1*/files/*
	if strings.Contains(path, "/files") {
		return true
	}

	// Cached contents: /v1*/cachedContents, /v1*/cachedContents/*
	if strings.Contains(path, "/cachedContents") {
		return true
	}

	// Corpora and semantic retrieval: /v1*/corpora, /v1*/corpora/*
	// Includes documents and chunks nested resources
	if strings.Contains(path, "/corpora") {
		return true
	}

	// Tuned models: /v1*/tunedModels, /v1*/tunedModels/*
	// Includes operations and permissions sub-resources
	if strings.Contains(path, "/tunedModels") {
		return true
	}

	// Batch operations: /v1*/batches, /v1*/batches/*
	if strings.Contains(path, "/batches") {
		return true
	}

	return false
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Identify provider early - use "unknown" as fallback for recording
	provider := p.identifyProvider(r.URL.Path)
	recordProvider := provider
	if recordProvider == "" {
		recordProvider = "unknown"
	}

	// Create recording FIRST - before ANY validation or body reading
	rec := recorder.NewRecording(recordProvider, r.Method, r.URL.Path, r.URL.RawQuery, startTime)
	rec.Request.Headers = r.Header.Clone()

	// Ensure recording happens even on early returns (including body read failures)
	defer func() {
		rec.Timing.CompletedAt = time.Now()
		rec.Timing.DurationMs = rec.Timing.CompletedAt.Sub(rec.Timing.StartedAt).Milliseconds()

		// Log completion
		logLevel := slog.LevelInfo
		if rec.Response.Status >= 400 {
			logLevel = slog.LevelError
		} else if rec.Response.Status >= 300 {
			logLevel = slog.LevelWarn
		}

		logAttrs := []any{
			"id", rec.ID[:8],
			"provider", rec.Provider,
			"status", rec.Response.Status,
			"duration_ms", rec.Timing.DurationMs,
			"path", rec.Request.Path,
		}
		if rec.Error != "" {
			logAttrs = append(logAttrs, "error", rec.Error)
		}

		slog.Log(r.Context(), logLevel, "request completed", logAttrs...)

		// Record asynchronously
		p.recorder.Record(rec)
	}()

	// Read and capture request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		rec.Error = "failed to read request body"
		rec.Response.Status = http.StatusBadRequest
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Capture request body
	if len(bodyBytes) > 0 {
		// Try to parse as JSON, otherwise store as string
		var jsonBody any
		if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
			rec.Request.Body = jsonBody
		} else {
			rec.Request.Body = string(bodyBytes)
		}
	}

	// Check if provider is known
	if provider == "" {
		rec.Error = "unknown API endpoint"
		rec.Response.Status = http.StatusNotFound
		slog.Warn("unknown API endpoint",
			"id", rec.ID[:8],
			"method", r.Method,
			"path", r.URL.Path,
			"host", r.Host,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.Header.Get("User-Agent"))
		http.Error(w, "unknown API endpoint", http.StatusNotFound)
		return
	}

	providerCfg, ok := p.cfg.Providers[provider]
	if !ok {
		rec.Error = fmt.Sprintf("provider %s not configured", provider)
		rec.Response.Status = http.StatusInternalServerError
		http.Error(w, rec.Error, http.StatusInternalServerError)
		return
	}

	// Create upstream request
	upstreamURL := providerCfg.UpstreamURL + r.URL.Path
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, bytes.NewReader(bodyBytes))
	if err != nil {
		rec.Error = "failed to create upstream request"
		rec.Response.Status = http.StatusInternalServerError
		http.Error(w, rec.Error, http.StatusInternalServerError)
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
		rec.Error = fmt.Sprintf("upstream request failed: %v", err)
		rec.Response.Status = http.StatusBadGateway
		slog.Error("upstream request failed", "id", rec.ID[:8], "error", err, "provider", provider, "path", r.URL.Path)
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
}

func (p *Proxy) handleRegular(w http.ResponseWriter, body io.Reader, rec *recorder.Recording) {
	var buf bytes.Buffer
	tee := io.TeeReader(body, &buf)

	if _, err := io.Copy(w, tee); err != nil {
		slog.Error("failed to copy response", "error", err)
		return
	}

	if buf.Len() > 0 {
		// Check if response is gzipped
		isGzipped := false
		if encodings, ok := rec.Response.Headers["Content-Encoding"]; ok {
			for _, encoding := range encodings {
				if strings.Contains(strings.ToLower(encoding), "gzip") {
					isGzipped = true
					break
				}
			}
		}

		// Try to parse as JSON, otherwise store as string or base64
		var jsonBody any
		if err := json.Unmarshal(buf.Bytes(), &jsonBody); err == nil {
			rec.Response.Body = jsonBody
		} else if isGzipped {
			// For gzipped content, base64 encode to preserve binary data
			rec.Response.Body = "base64:" + base64.StdEncoding.EncodeToString(buf.Bytes())
		} else {
			rec.Response.Body = buf.String()
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
