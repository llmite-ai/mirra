package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/jpoz/mirra/internal/server"
)

func TestMirraAlive_Detection(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    bool
	}{
		{
			name: "mirra health endpoint",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(server.HeaderMirra, "1")
				w.WriteHeader(http.StatusOK)
			},
			want: true,
		},
		{
			name: "unrelated server without mirra header",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			want: false,
		},
		{
			name: "server erroring",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(server.HeaderMirra, "1")
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(tt.handler)
			defer ts.Close()
			if got := mirraAlive(ts.URL); got != tt.want {
				t.Fatalf("mirraAlive() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestMirraAlive_ServerDown(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	ts.Close()
	if mirraAlive(ts.URL) {
		t.Fatal("mirraAlive() = true for a closed server, want false")
	}
}

func TestClaudeEnv_Scenarios(t *testing.T) {
	tests := []struct {
		name string
		base []string
		want []string
	}{
		{
			name: "appends when absent",
			base: []string{"HOME=/home/x", "PATH=/bin"},
			want: []string{"HOME=/home/x", "PATH=/bin", "ANTHROPIC_BASE_URL=http://localhost:4567"},
		},
		{
			name: "replaces existing value",
			base: []string{"ANTHROPIC_BASE_URL=https://gateway.example.com", "PATH=/bin"},
			want: []string{"PATH=/bin", "ANTHROPIC_BASE_URL=http://localhost:4567"},
		},
		{
			name: "leaves similar names alone",
			base: []string{"ANTHROPIC_BASE_URL_BACKUP=https://x.example.com"},
			want: []string{"ANTHROPIC_BASE_URL_BACKUP=https://x.example.com", "ANTHROPIC_BASE_URL=http://localhost:4567"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := claudeEnv(tt.base, "http://localhost:4567")
			if !slices.Equal(got, tt.want) {
				t.Fatalf("claudeEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHoldSession_HoldsUntilReleased(t *testing.T) {
	held := make(chan struct{}, 1)
	released := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(server.HeaderMirra, "1")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		held <- struct{}{}
		<-r.Context().Done()
		close(released)
	}))
	defer ts.Close()

	release, err := holdSession(ts.URL)
	if err != nil {
		t.Fatalf("holdSession() = %v, want nil", err)
	}
	select {
	case <-held:
	case <-time.After(2 * time.Second):
		t.Fatal("server never saw the hold request")
	}

	release()
	select {
	case <-released:
	case <-time.After(2 * time.Second):
		t.Fatal("release() did not disconnect the hold request")
	}
}

func TestHoldSession_RejectsNonMirraResponse(t *testing.T) {
	// A mirra predating the hold endpoint answers via the SPA fallback:
	// 200 with HTML and no mirra header.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer ts.Close()

	if _, err := holdSession(ts.URL); err == nil {
		t.Fatal("holdSession() = nil error for a server without hold support, want error")
	}
}

func TestWaitMirraAlive_EventualSuccess(t *testing.T) {
	var ready bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			ready = true
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set(server.HeaderMirra, "1")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan bool, 1)
	go func() { done <- waitMirraAlive(ts.URL, 5) }()
	select {
	case got := <-done:
		if !got {
			t.Fatal("waitMirraAlive() = false, want true after server becomes ready")
		}
	case <-ctx.Done():
		t.Fatal("waitMirraAlive() did not return in time")
	}
}
