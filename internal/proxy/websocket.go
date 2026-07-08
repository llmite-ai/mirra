package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jpoz/mirra/internal/recorder"
)

// WebSocket frame opcodes (RFC 6455 §5.2).
const (
	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
)

const (
	directionSent     = "sent"     // client → upstream
	directionReceived = "received" // upstream → client
)

// IsWebSocketUpgrade reports whether the request asks to upgrade the
// connection to a WebSocket (RFC 6455 opening handshake).
func IsWebSocketUpgrade(r *http.Request) bool {
	if r.Method != http.MethodGet || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}
	for _, v := range r.Header.Values("Connection") {
		for _, token := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
				return true
			}
		}
	}
	return false
}

// handleWebSocket tunnels a WebSocket connection to the provider upstream.
// Codex uses this transport for /v1/responses. The client's handshake is
// forwarded nearly verbatim over a raw TCP/TLS connection, then frames are
// spliced byte for byte in both directions while complete text and binary
// messages are decoded into the recording. A non-101 upstream answer is
// relayed unchanged so clients (codex included) can fall back to their HTTP
// transport.
func (p *Proxy) handleWebSocket(w http.ResponseWriter, r *http.Request, provider, forwardPath string, rec *recorder.Recording) {
	providerCfg, ok := p.cfg.Providers[provider]
	if !ok {
		rec.Error = fmt.Sprintf("provider %s not configured", provider)
		rec.Response.Status = http.StatusInternalServerError
		http.Error(w, rec.Error, http.StatusInternalServerError)
		return
	}

	upstreamURL := providerCfg.UpstreamURL + forwardPath
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}
	u, err := url.Parse(upstreamURL)
	if err != nil {
		rec.Error = fmt.Sprintf("invalid upstream URL: %v", err)
		rec.Response.Status = http.StatusInternalServerError
		http.Error(w, "invalid upstream URL", http.StatusInternalServerError)
		return
	}

	upstream, err := dialUpstream(u)
	if err != nil {
		rec.Error = fmt.Sprintf("upstream dial failed: %v", err)
		rec.Response.Status = http.StatusBadGateway
		slog.Error("websocket upstream dial failed", "id", rec.ID[:8], "error", err, "provider", provider)
		http.Error(w, "upstream dial failed", http.StatusBadGateway)
		return
	}
	defer func() {
		_ = upstream.Close()
	}()

	if err := writeHandshake(upstream, r, u); err != nil {
		rec.Error = fmt.Sprintf("upstream handshake write failed: %v", err)
		rec.Response.Status = http.StatusBadGateway
		http.Error(w, "upstream handshake failed", http.StatusBadGateway)
		return
	}

	upstreamReader := bufio.NewReader(upstream)
	resp, err := http.ReadResponse(upstreamReader, r)
	if err != nil {
		rec.Error = fmt.Sprintf("upstream handshake read failed: %v", err)
		rec.Response.Status = http.StatusBadGateway
		http.Error(w, "upstream handshake failed", http.StatusBadGateway)
		return
	}

	rec.Response.Status = resp.StatusCode
	rec.Response.Headers = resp.Header.Clone()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		// Relay the refusal unchanged; codex falls back to its HTTP
		// transport on anything but a 101.
		defer func() {
			_ = resp.Body.Close()
		}()
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		p.handleRegular(w, resp.Body, rec)
		return
	}

	hj, ok := w.(http.Hijacker)
	if !ok {
		rec.Error = "connection does not support hijacking for websocket upgrade"
		rec.Response.Status = http.StatusInternalServerError
		http.Error(w, rec.Error, http.StatusInternalServerError)
		return
	}
	clientConn, clientRW, err := hj.Hijack()
	if err != nil {
		rec.Error = fmt.Sprintf("hijack failed: %v", err)
		rec.Response.Status = http.StatusInternalServerError
		http.Error(w, "hijack failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = clientConn.Close()
	}()
	// The tunnel outlives any server read/write deadlines.
	_ = clientConn.SetDeadline(time.Time{})

	// Relay the 101 verbatim so the Sec-WebSocket-Accept the upstream
	// computed reaches the client that sent the matching key.
	if err := writeHandshakeResponse(clientConn, resp); err != nil {
		rec.Error = fmt.Sprintf("failed to relay 101 to client: %v", err)
		return
	}

	rec.Response.Streaming = true
	slog.Info("websocket tunnel established", "id", rec.ID[:8], "provider", provider, "path", rec.Request.Path)

	// Splice both directions. Closing both conns when either copier exits
	// unblocks the peer copier, so the tunnel tears down as one unit.
	msgs := &messageLog{}
	closeBoth := func() {
		_ = upstream.Close()
		_ = clientConn.Close()
	}
	var received int64
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer closeBoth()
		copyFrames(upstream, clientRW.Reader, directionSent, msgs, rec.ID)
	}()
	go func() {
		defer wg.Done()
		defer closeBoth()
		received = copyFrames(clientConn, upstreamReader, directionReceived, msgs, rec.ID)
	}()
	wg.Wait()

	rec.ResponseSize = received
	rec.Response.Body = msgs.snapshot()
}

// dialUpstream opens the raw connection the tunnel runs over. TLS pins
// HTTP/1.1 because a WebSocket upgrade cannot ride HTTP/2.
func dialUpstream(u *url.URL) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	switch u.Scheme {
	case "https", "wss":
		host := u.Host
		if u.Port() == "" {
			host = net.JoinHostPort(u.Hostname(), "443")
		}
		return tls.DialWithDialer(dialer, "tcp", host, &tls.Config{
			ServerName: u.Hostname(),
			NextProtos: []string{"http/1.1"},
		})
	default:
		host := u.Host
		if u.Port() == "" {
			host = net.JoinHostPort(u.Hostname(), "80")
		}
		return dialer.Dial("tcp", host)
	}
}

// writeHandshake forwards the client's upgrade request to the upstream.
// Sec-WebSocket-Extensions is dropped so no compression is negotiated and
// the frames crossing the tunnel stay readable for the recording;
// Accept-Encoding is dropped for the same reason on a refused upgrade.
func writeHandshake(conn net.Conn, r *http.Request, u *url.URL) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "GET %s HTTP/1.1\r\n", u.RequestURI())
	fmt.Fprintf(&b, "Host: %s\r\n", u.Host)
	for key, values := range r.Header {
		if strings.EqualFold(key, "Sec-Websocket-Extensions") || strings.EqualFold(key, "Accept-Encoding") {
			continue
		}
		for _, value := range values {
			fmt.Fprintf(&b, "%s: %s\r\n", key, value)
		}
	}
	b.WriteString("\r\n")
	_, err := conn.Write(b.Bytes())
	return err
}

// writeHandshakeResponse relays the upstream's 101 status line and headers
// to the hijacked client connection.
func writeHandshakeResponse(conn net.Conn, resp *http.Response) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "HTTP/1.1 %s\r\n", resp.Status)
	if err := resp.Header.Write(&b); err != nil {
		return err
	}
	b.WriteString("\r\n")
	_, err := conn.Write(b.Bytes())
	return err
}

// wsMessage is one complete WebSocket message as it crossed the tunnel.
type wsMessage struct {
	Direction string    `json:"direction"`
	At        time.Time `json:"at"`
	Type      string    `json:"type"` // "text" or "binary"
	Data      any       `json:"data,omitempty"`
}

// messageLog collects messages from both tunnel directions.
type messageLog struct {
	mu   sync.Mutex
	msgs []wsMessage
}

func (l *messageLog) add(direction, msgType string, payload []byte) {
	msg := wsMessage{Direction: direction, At: time.Now(), Type: msgType}
	if len(payload) > 0 {
		var jsonBody any
		if err := json.Unmarshal(payload, &jsonBody); err == nil {
			msg.Data = jsonBody
		} else {
			msg.Data = recordString(payload)
		}
	}
	l.mu.Lock()
	l.msgs = append(l.msgs, msg)
	l.mu.Unlock()
}

func (l *messageLog) snapshot() []wsMessage {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]wsMessage(nil), l.msgs...)
}

// copyFrames forwards WebSocket frames from src to dst byte for byte,
// decoding complete text and binary messages into log along the way. Client
// frames are masked on the wire (RFC 6455 §5.3); the mask is undone only on
// the recorded copy, never on the forwarded bytes. Payloads stream through
// in chunks so a frame's declared size is never trusted for an allocation.
// Returns the number of payload bytes forwarded. Read/write errors end the
// copy silently: tunnel teardown closes both conns, so errors here are the
// expected shutdown path.
func copyFrames(dst io.Writer, src *bufio.Reader, direction string, log *messageLog, recID string) int64 {
	var total int64
	var message bytes.Buffer
	var messageType string
	header := make([]byte, 2)
	ext := make([]byte, 8)
	maskKey := make([]byte, 4)
	buf := make([]byte, 32*1024)

	for {
		if _, err := io.ReadFull(src, header); err != nil {
			return total
		}
		fin := header[0]&0x80 != 0
		opcode := header[0] & 0x0f
		masked := header[1]&0x80 != 0
		length := int64(header[1] & 0x7f)

		extLen := 0
		switch length {
		case 126:
			extLen = 2
		case 127:
			extLen = 8
		}
		if extLen > 0 {
			if _, err := io.ReadFull(src, ext[:extLen]); err != nil {
				return total
			}
			if extLen == 2 {
				length = int64(binary.BigEndian.Uint16(ext[:2]))
			} else {
				length = int64(binary.BigEndian.Uint64(ext[:8]))
			}
			if length < 0 {
				slog.Warn("websocket frame length overflow, closing tunnel", "id", recID[:8], "direction", direction)
				return total
			}
		}
		if masked {
			if _, err := io.ReadFull(src, maskKey); err != nil {
				return total
			}
		}

		// Forward the header exactly as read.
		if _, err := dst.Write(header); err != nil {
			return total
		}
		if extLen > 0 {
			if _, err := dst.Write(ext[:extLen]); err != nil {
				return total
			}
		}
		if masked {
			if _, err := dst.Write(maskKey); err != nil {
				return total
			}
		}

		// Control frames (opcode >= 8) may interleave with a fragmented
		// message and are forwarded but not recorded.
		record := opcode == opText || opcode == opBinary ||
			(opcode == opContinuation && messageType != "")
		switch opcode {
		case opText:
			messageType = "text"
		case opBinary:
			messageType = "binary"
		}

		logged := false
		var offset int64
		remaining := length
		for remaining > 0 {
			n := len(buf)
			if remaining < int64(n) {
				n = int(remaining)
			}
			m, err := src.Read(buf[:n])
			if m > 0 {
				chunk := buf[:m]
				remaining -= int64(m)
				if record {
					start := message.Len()
					message.Write(chunk)
					if masked {
						written := message.Bytes()[start:]
						for i := range written {
							written[i] ^= maskKey[(offset+int64(i))%4]
						}
					}
					// A completed message is logged before its final byte
					// is forwarded, so a recorded reply can never precede
					// the message that provoked it.
					if fin && remaining == 0 {
						log.add(direction, messageType, message.Bytes())
						logged = true
					}
				}
				if _, werr := dst.Write(chunk); werr != nil {
					return total
				}
				total += int64(m)
				offset += int64(m)
			}
			if err != nil {
				return total
			}
		}

		if record && fin {
			if !logged { // empty final frame: nothing streamed through the loop
				log.add(direction, messageType, message.Bytes())
			}
			message.Reset()
			messageType = ""
		}
	}
}
