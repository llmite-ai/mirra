package proxy

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/recorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wsAcceptKey(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h[:])
}

// writeFrame writes a single WebSocket frame, choosing the 16- or 64-bit
// extended length form as the payload requires.
func writeFrame(t *testing.T, w io.Writer, fin bool, opcode byte, payload []byte, mask bool) {
	t.Helper()

	b0 := opcode
	if fin {
		b0 |= 0x80
	}
	var header []byte
	switch l := len(payload); {
	case l < 126:
		header = []byte{b0, byte(l)}
	case l <= 0xffff:
		header = []byte{b0, 126, 0, 0}
		binary.BigEndian.PutUint16(header[2:], uint16(l))
	default:
		header = append([]byte{b0, 127}, make([]byte, 8)...)
		binary.BigEndian.PutUint64(header[2:], uint64(l))
	}
	if mask {
		header[1] |= 0x80
		key := []byte{0x11, 0x22, 0x33, 0x44}
		masked := make([]byte, len(payload))
		for i, c := range payload {
			masked[i] = c ^ key[i%4]
		}
		header = append(header, key...)
		payload = masked
	}
	_, err := w.Write(header)
	require.NoError(t, err)
	_, err = w.Write(payload)
	require.NoError(t, err)
}

// readTestFrame reads one frame, unmasking the payload if needed.
func readTestFrame(t *testing.T, br *bufio.Reader) (opcode byte, fin bool, payload []byte) {
	t.Helper()

	header := make([]byte, 2)
	_, err := io.ReadFull(br, header)
	require.NoError(t, err)
	fin = header[0]&0x80 != 0
	opcode = header[0] & 0x0f
	masked := header[1]&0x80 != 0
	length := int64(header[1] & 0x7f)
	switch length {
	case 126:
		ext := make([]byte, 2)
		_, err = io.ReadFull(br, ext)
		require.NoError(t, err)
		length = int64(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		_, err = io.ReadFull(br, ext)
		require.NoError(t, err)
		length = int64(binary.BigEndian.Uint64(ext))
	}
	var key []byte
	if masked {
		key = make([]byte, 4)
		_, err = io.ReadFull(br, key)
		require.NoError(t, err)
	}
	payload = make([]byte, length)
	_, err = io.ReadFull(br, payload)
	require.NoError(t, err)
	if masked {
		for i := range payload {
			payload[i] ^= key[i%4]
		}
	}
	return opcode, fin, payload
}

// readTestMessage assembles one complete data message, following
// continuation frames until FIN.
func readTestMessage(t *testing.T, br *bufio.Reader) []byte {
	t.Helper()

	var message []byte
	for {
		_, fin, payload := readTestFrame(t, br)
		message = append(message, payload...)
		if fin {
			return message
		}
	}
}

// A codex websocket session must tunnel through the proxy: the upgrade
// reaches the chatgpt upstream with the /v1 prefix dropped and no
// compression extension, frames pass in both directions (masked, fragmented,
// and extended-length ones included), and the recording holds every message
// as structured JSON.
func TestHandle_WebSocketTunnel(t *testing.T) {
	// Big enough to force the 64-bit extended length form.
	bigPadding := strings.Repeat("x", 70_000)
	upstreamEvent := `{"type":"response.completed","padding":"` + bigPadding + `"}`

	var gotPath, gotExtensions, gotClientMessage string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotExtensions = r.Header.Get("Sec-WebSocket-Extensions")

		conn, rw, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		defer func() {
			_ = conn.Close()
		}()

		_, err = fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
			wsAcceptKey(r.Header.Get("Sec-WebSocket-Key")))
		require.NoError(t, err)

		gotClientMessage = string(readTestMessage(t, rw.Reader))
		writeFrame(t, conn, true, opText, []byte(upstreamEvent), false)
		writeFrame(t, conn, true, 0x8, []byte{0x03, 0xe8}, false) // close, code 1000
	}))
	defer upstream.Close()

	dir := t.TempDir()
	rec := recorder.New(true, dir)
	p := New(&config.Config{
		Providers: map[string]config.Provider{
			"chatgpt": {UpstreamURL: upstream.URL + "/backend-api/codex"},
		},
	}, rec)

	handleDone := make(chan struct{})
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.Handle(w, r)
		close(handleDone)
	}))
	defer proxySrv.Close()

	conn, err := net.Dial("tcp", proxySrv.Listener.Addr().String())
	require.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	clientKey := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	handshake := "GET /v1/responses HTTP/1.1\r\n" +
		"Host: " + proxySrv.Listener.Addr().String() + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + clientKey + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Extensions: permessage-deflate\r\n" +
		"ChatGPT-Account-ID: acct-123\r\n" +
		"\r\n"
	_, err = conn.Write([]byte(handshake))
	require.NoError(t, err)

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	// The upstream computed the accept from the key the client sent, and the
	// proxy relayed it untouched.
	assert.Equal(t, wsAcceptKey(clientKey), resp.Header.Get("Sec-Websocket-Accept"))

	// Send one text message fragmented across two masked frames.
	writeFrame(t, conn, false, opText, []byte(`{"model":`), true)
	writeFrame(t, conn, true, opContinuation, []byte(`"gpt-5.5"}`), true)

	got := readTestMessage(t, br)
	assert.Equal(t, upstreamEvent, string(got))

	opcode, _, closePayload := readTestFrame(t, br)
	assert.Equal(t, byte(0x8), opcode)
	assert.Equal(t, []byte{0x03, 0xe8}, closePayload)

	select {
	case <-handleDone:
	case <-time.After(5 * time.Second):
		t.Fatal("proxy handler did not finish after upstream closed")
	}

	assert.Equal(t, "/backend-api/codex/responses", gotPath)
	assert.Empty(t, gotExtensions, "compression negotiation must not reach the upstream")
	assert.JSONEq(t, `{"model":"gpt-5.5"}`, gotClientMessage)

	recording := readRecordedBody(t, rec, dir)
	assert.Equal(t, "chatgpt", recording.Provider)
	assert.Equal(t, http.StatusSwitchingProtocols, recording.Response.Status)
	assert.True(t, recording.Response.Streaming)

	messages, ok := recording.Response.Body.([]any)
	require.True(t, ok, "expected recorded message list, got %T", recording.Response.Body)
	require.Len(t, messages, 2)

	sent := messages[0].(map[string]any)
	assert.Equal(t, "sent", sent["direction"])
	assert.Equal(t, "text", sent["type"])
	assert.Equal(t, map[string]any{"model": "gpt-5.5"}, sent["data"])

	received := messages[1].(map[string]any)
	assert.Equal(t, "received", received["direction"])
	assert.Equal(t, "text", received["type"])
	assert.Equal(t, "response.completed", received["data"].(map[string]any)["type"])
}

// An upstream that refuses the upgrade must have its answer relayed
// unchanged (codex falls back to HTTP streaming on any non-101) and
// recorded.
func TestHandle_WebSocketUpstreamRefusalRelayed(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"detail":"websockets not enabled"}`))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	rec := recorder.New(true, dir)
	p := New(&config.Config{
		Providers: map[string]config.Provider{
			"chatgpt": {UpstreamURL: upstream.URL + "/backend-api/codex"},
		},
	}, rec)

	req := httptest.NewRequest(http.MethodGet, "/v1/responses", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", base64.StdEncoding.EncodeToString([]byte("0123456789abcdef")))
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("ChatGPT-Account-ID", "acct-123")
	w := httptest.NewRecorder()
	p.Handle(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.JSONEq(t, `{"detail":"websockets not enabled"}`, w.Body.String())

	recording := readRecordedBody(t, rec, dir)
	assert.Equal(t, "chatgpt", recording.Provider)
	assert.Equal(t, http.StatusForbidden, recording.Response.Status)
	assert.Equal(t, map[string]any{"detail": "websockets not enabled"}, recording.Response.Body)
}

func TestIsWebSocketUpgrade(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		upgrade    string
		connection string
		want       bool
	}{
		{"codex upgrade", http.MethodGet, "websocket", "Upgrade", true},
		{"connection token list", http.MethodGet, "WebSocket", "keep-alive, Upgrade", true},
		{"plain get", http.MethodGet, "", "", false},
		{"post with upgrade headers", http.MethodPost, "websocket", "Upgrade", false},
		{"upgrade without connection", http.MethodGet, "websocket", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/v1/responses", nil)
			if tt.upgrade != "" {
				r.Header.Set("Upgrade", tt.upgrade)
			}
			if tt.connection != "" {
				r.Header.Set("Connection", tt.connection)
			}
			assert.Equal(t, tt.want, IsWebSocketUpgrade(r))
		})
	}
}
