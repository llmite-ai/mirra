package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/klauspost/compress/zstd"

	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/recorder"
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
	// Gemini uses colons for operations (e.g., :generateContent)
	// Only match if path contains both "/models" and ":" to avoid conflicts with OpenAI /v1/models/{id}
	if (strings.Contains(path, "/models/") || strings.Contains(path, "/models:")) && strings.Contains(path, ":") {
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
	forwardPath := r.URL.Path

	// Codex sessions authenticated with a ChatGPT subscription use
	// OpenAI-shaped paths but a different backend; the account header is what
	// tells the two apart. The /v1 prefix exists only on the OpenAI API side,
	// so it is dropped before joining with the chatgpt upstream.
	if provider == "openai" && r.Header.Get("ChatGPT-Account-ID") != "" {
		provider = "chatgpt"
		forwardPath = strings.TrimPrefix(r.URL.Path, "/v1")
	}

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

	// WebSocket upgrades (codex's transport for /v1/responses) cannot ride an
	// http.Client round trip; they get a raw tunnel that records the frames
	// as they pass. Unknown endpoints fall through to the 404 below.
	if provider != "" && IsWebSocketUpgrade(r) {
		p.handleWebSocket(w, r, provider, forwardPath, &rec)
		return
	}

	// Read and capture request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		rec.Error = "failed to read request body"
		rec.Response.Status = http.StatusBadRequest
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	// Capture request body. Some clients compress request payloads (Codex
	// sends zstd); the recording holds the decoded form while the upstream
	// still receives the original bytes.
	if len(bodyBytes) > 0 {
		decoded := decodeBody(r.Header, bodyBytes)
		var jsonBody any
		if err := json.Unmarshal(decoded, &jsonBody); err == nil {
			rec.Request.Body = jsonBody
		} else {
			rec.Request.Body = recordString(decoded)
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
	upstreamURL := providerCfg.UpstreamURL + forwardPath
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

	// Copy headers. Accept-Encoding is deliberately not forwarded: Go's
	// transport then negotiates gzip itself and transparently decompresses,
	// so the client and the recording both see identity-encoded bytes while
	// the upstream leg stays compressed.
	for key, values := range r.Header {
		if strings.EqualFold(key, "Accept-Encoding") {
			continue
		}
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
	defer func() {
		_ = resp.Body.Close()
	}()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	rec.Response.Status = resp.StatusCode
	rec.Response.Headers = resp.Header.Clone()

	// Check if streaming. The chatgpt upstream omits Content-Type on its SSE
	// responses, so the client's Accept header is the only remaining signal.
	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "text/event-stream") ||
		strings.Contains(contentType, "stream")
	if contentType == "" && resp.StatusCode < 300 &&
		strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		isStreaming = true
	}
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
		slog.Error("failed to copy response", "id", rec.ID[:8], "error", err)
		return
	}

	// Set response size
	rec.ResponseSize = int64(buf.Len())

	if buf.Len() > 0 {
		decoded := decodeBody(rec.Response.Headers, buf.Bytes())

		// Try to parse as JSON, otherwise store as text or base64
		var jsonBody any
		if err := json.Unmarshal(decoded, &jsonBody); err == nil {
			rec.Response.Body = jsonBody
		} else {
			rec.Response.Body = recordString(decoded)
		}
	}
}

// decodeBody reverses a gzip or zstd Content-Encoding so recordings hold
// readable payloads even when a peer compresses without being asked.
// Undecodable input is returned unchanged; a truncated stream (client
// disconnect mid-response) still yields the prefix that was flushed.
func decodeBody(headers map[string][]string, raw []byte) []byte {
	switch {
	case hasContentEncoding(headers, "gzip"):
		gz, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return raw
		}
		defer func() {
			_ = gz.Close()
		}()
		decoded, err := io.ReadAll(gz)
		if err != nil && len(decoded) == 0 {
			return raw
		}
		return decoded
	case hasContentEncoding(headers, "zstd"):
		zr, err := zstd.NewReader(bytes.NewReader(raw))
		if err != nil {
			return raw
		}
		defer zr.Close()
		decoded, err := io.ReadAll(zr)
		if err != nil && len(decoded) == 0 {
			return raw
		}
		return decoded
	}
	return raw
}

func hasContentEncoding(headers map[string][]string, name string) bool {
	for _, encoding := range headers["Content-Encoding"] {
		if strings.Contains(strings.ToLower(encoding), name) {
			return true
		}
	}
	return false
}

// recordString stores text bodies as strings; binary bodies are base64
// encoded because encoding/json replaces invalid UTF-8 with U+FFFD, which
// would destroy the payload.
func recordString(b []byte) any {
	if utf8.Valid(b) {
		return string(b)
	}
	return "base64:" + base64.StdEncoding.EncodeToString(b)
}

func (p *Proxy) handleStreaming(w http.ResponseWriter, body io.Reader, rec *recorder.Recording) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("response writer does not support flushing", "id", rec.ID[:8])
		p.handleRegular(w, body, rec)
		return
	}

	// Copy raw chunks as they arrive: a line scanner would strip \r bytes,
	// hold data back until a newline shows up, and abort the stream entirely
	// on lines longer than its buffer.
	var accumulated bytes.Buffer
	buf := make([]byte, 32*1024)
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			accumulated.Write(chunk)
			if _, writeErr := w.Write(chunk); writeErr != nil {
				slog.Error("failed to write streaming chunk", "id", rec.ID[:8], "error", writeErr)
				break
			}
			flusher.Flush()
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			slog.Error("error reading stream", "id", rec.ID[:8], "error", readErr)
			break
		}
	}

	// Set response size
	rec.ResponseSize = int64(accumulated.Len())

	if accumulated.Len() > 0 {
		// SSE payloads stay a string so the stream parse endpoint can read
		// them; gzip is reversed and binary falls back to base64.
		rec.Response.Body = recordString(decodeBody(rec.Response.Headers, accumulated.Bytes()))
	}
}
