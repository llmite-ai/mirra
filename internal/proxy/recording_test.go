package proxy

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"

	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readRecordedBody flushes the recorder and returns the single recording
// written during the test.
func readRecordedBody(t *testing.T, rec *recorder.Recorder, dir string) recorder.Recording {
	t.Helper()
	require.NoError(t, rec.Close())

	matches, err := filepath.Glob(filepath.Join(dir, "recordings-*.jsonl"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	f, err := os.Open(matches[0])
	require.NoError(t, err)
	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	require.True(t, scanner.Scan(), "expected one recording line")

	var recording recorder.Recording
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &recording))
	return recording
}

// Upstreams may gzip SSE responses. The recording must hold the decompressed
// event text (raw gzip bytes stored as a string are destroyed by JSON's
// UTF-8 replacement), and the client must receive a stream it can read.
func TestHandle_GzipStreamingResponseRecordedAsText(t *testing.T) {
	sse := "event: message_start\r\ndata: {\"type\":\"message_start\"}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	var upstreamAcceptEncoding string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamAcceptEncoding = r.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, err := gz.Write([]byte(sse))
		require.NoError(t, err)
		require.NoError(t, gz.Close())
	}))
	defer upstream.Close()

	dir := t.TempDir()
	rec := recorder.New(true, dir)
	p := New(&config.Config{
		Providers: map[string]config.Provider{
			"claude": {UpstreamURL: upstream.URL},
		},
	}, rec)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude"}`))
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	w := httptest.NewRecorder()
	p.Handle(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	// The client's Accept-Encoding must not reach the upstream; the transport
	// negotiates gzip itself so it can transparently decompress.
	assert.NotContains(t, upstreamAcceptEncoding, "zstd")
	// The client receives the decoded stream.
	assert.Equal(t, sse, w.Body.String())

	recording := readRecordedBody(t, rec, dir)
	assert.True(t, recording.Response.Streaming)
	assert.Equal(t, sse, recording.Response.Body)
}

// Gzipped non-streaming JSON must be recorded as structured JSON, not base64.
func TestHandle_GzipJSONResponseRecordedAsJSON(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, err := gz.Write([]byte(`{"id":"msg_123","role":"assistant"}`))
		require.NoError(t, err)
		require.NoError(t, gz.Close())
	}))
	defer upstream.Close()

	dir := t.TempDir()
	rec := recorder.New(true, dir)
	p := New(&config.Config{
		Providers: map[string]config.Provider{
			"claude": {UpstreamURL: upstream.URL},
		},
	}, rec)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("Accept-Encoding", "gzip, br")
	w := httptest.NewRecorder()
	p.Handle(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"id":"msg_123","role":"assistant"}`, w.Body.String())

	recording := readRecordedBody(t, rec, dir)
	assert.Equal(t, map[string]any{"id": "msg_123", "role": "assistant"}, recording.Response.Body)
}

// SSE events can carry data lines far larger than a line scanner's buffer;
// the stream must be forwarded byte for byte without truncating.
func TestHandle_StreamingLargeLineForwardedIntact(t *testing.T) {
	payload := "event: content_block_delta\ndata: {\"text\":\"" + strings.Repeat("x", 2*1024*1024) + "\"}\n\n"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, err := w.Write([]byte(payload))
		require.NoError(t, err)
	}))
	defer upstream.Close()

	p := New(&config.Config{
		Providers: map[string]config.Provider{
			"claude": {UpstreamURL: upstream.URL},
		},
	}, recorder.New(false, ""))

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	p.Handle(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, payload, w.Body.String())
}

// Codex compresses request bodies with zstd. The recording must hold the
// decoded JSON (raw zstd bytes stored as a string are destroyed by JSON's
// UTF-8 replacement), while the upstream still receives the original
// compressed bytes.
func TestHandle_ZstdRequestBodyRecordedDecoded(t *testing.T) {
	payload := `{"model":"gpt-5.5","stream":true}`
	var buf bytes.Buffer
	zw, err := zstd.NewWriter(&buf)
	require.NoError(t, err)
	_, err = zw.Write([]byte(payload))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	compressed := buf.Bytes()

	var upstreamBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		require.NoError(t, readErr)
		upstreamBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	rec := recorder.New(true, dir)
	p := New(&config.Config{
		Providers: map[string]config.Provider{
			"chatgpt": {UpstreamURL: upstream.URL},
		},
	}, rec)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(compressed))
	req.Header.Set("ChatGPT-Account-ID", "acct-123")
	req.Header.Set("Content-Encoding", "zstd")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	p.Handle(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	// The upstream still receives the compressed bytes untouched.
	assert.Equal(t, compressed, upstreamBody)

	recording := readRecordedBody(t, rec, dir)
	assert.Equal(t, map[string]any{"model": "gpt-5.5", "stream": true}, recording.Request.Body)
}

// The chatgpt upstream sends SSE without a Content-Type header; the request's
// Accept header is the only signal that the response is a stream.
func TestHandle_MissingContentTypeSSEDetectedViaAccept(t *testing.T) {
	sse := "event: response.created\ndata: {\"type\":\"response.created\"}\n\n"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No Content-Type set; Go would sniff one, so explicitly delete it.
		w.Header()["Content-Type"] = nil
		_, _ = w.Write([]byte(sse))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	rec := recorder.New(true, dir)
	p := New(&config.Config{
		Providers: map[string]config.Provider{
			"chatgpt": {UpstreamURL: upstream.URL},
		},
	}, rec)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"stream":true}`))
	req.Header.Set("ChatGPT-Account-ID", "acct-123")
	req.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()
	p.Handle(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, sse, w.Body.String())

	recording := readRecordedBody(t, rec, dir)
	assert.True(t, recording.Response.Streaming)
	assert.Equal(t, sse, recording.Response.Body)
}

func TestDecodeBody(t *testing.T) {
	gzipped := func(s string) []byte {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write([]byte(s))
		require.NoError(t, err)
		require.NoError(t, gz.Close())
		return buf.Bytes()
	}
	zstded := func(s string) []byte {
		var buf bytes.Buffer
		zw, err := zstd.NewWriter(&buf)
		require.NoError(t, err)
		_, err = zw.Write([]byte(s))
		require.NoError(t, err)
		require.NoError(t, zw.Close())
		return buf.Bytes()
	}

	tests := []struct {
		name    string
		headers map[string][]string
		raw     []byte
		want    []byte
	}{
		{
			name:    "gzip encoded is decompressed",
			headers: map[string][]string{"Content-Encoding": {"gzip"}},
			raw:     gzipped("data: hello\n\n"),
			want:    []byte("data: hello\n\n"),
		},
		{
			name:    "no encoding passes through",
			headers: map[string][]string{},
			raw:     []byte("data: hello\n\n"),
			want:    []byte("data: hello\n\n"),
		},
		{
			name:    "invalid gzip returns raw bytes",
			headers: map[string][]string{"Content-Encoding": {"gzip"}},
			raw:     []byte("not gzip"),
			want:    []byte("not gzip"),
		},
		{
			name:    "zstd encoded is decompressed",
			headers: map[string][]string{"Content-Encoding": {"zstd"}},
			raw:     zstded("data: hello\n\n"),
			want:    []byte("data: hello\n\n"),
		},
		{
			name:    "invalid zstd returns raw bytes",
			headers: map[string][]string{"Content-Encoding": {"zstd"}},
			raw:     []byte("not zstd"),
			want:    []byte("not zstd"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, decodeBody(tt.headers, tt.raw))
		})
	}
}

func TestRecordString(t *testing.T) {
	assert.Equal(t, "data: hi\n\n", recordString([]byte("data: hi\n\n")))

	binary := []byte{0x1f, 0x8b, 0xff, 0xfe, 0x00}
	got, ok := recordString(binary).(string)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(got, "base64:"))
}
