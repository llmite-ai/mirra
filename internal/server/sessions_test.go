package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionTracker_WaitDrained(t *testing.T) {
	tests := []struct {
		name  string
		holds int
	}{
		{name: "already empty", holds: 0},
		{name: "single hold", holds: 1},
		{name: "multiple holds", holds: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newSessionTracker()
			for range tt.holds {
				tr.add(1)
			}
			if got := tr.Count(); got != tt.holds {
				t.Fatalf("Count() = %d, want %d", got, tt.holds)
			}

			go func() {
				for range tt.holds {
					tr.add(-1)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := tr.WaitDrained(ctx); err != nil {
				t.Fatalf("WaitDrained() = %v, want nil", err)
			}
			if got := tr.Count(); got != 0 {
				t.Fatalf("Count() after drain = %d, want 0", got)
			}
		})
	}
}

func TestSessionTracker_WaitDrainedCanceled(t *testing.T) {
	tr := newSessionTracker()
	tr.add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := tr.WaitDrained(ctx); err == nil {
		t.Fatal("WaitDrained() = nil with a held session, want context error")
	}
}

func TestHoldHandler_CountsUntilDisconnect(t *testing.T) {
	s := &Server{
		log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		held:    newSessionTracker(),
		closing: make(chan struct{}),
	}
	ts := httptest.NewServer(http.HandlerFunc(s.holdHandler))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("hold request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resp.Header.Get(HeaderMirra) == "" {
		t.Fatalf("missing %s header on hold response", HeaderMirra)
	}
	// The count increments before response headers are written, so it must
	// be visible as soon as Do returns.
	if got := s.held.Count(); got != 1 {
		t.Fatalf("Count() while held = %d, want 1", got)
	}

	cancel()
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()
	if err := s.held.WaitDrained(waitCtx); err != nil {
		t.Fatalf("hold was not released after client disconnect: %v", err)
	}
}

func TestHoldHandler_ReleasedOnServerClose(t *testing.T) {
	s := &Server{
		log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		held:    newSessionTracker(),
		closing: make(chan struct{}),
	}
	ts := httptest.NewServer(http.HandlerFunc(s.holdHandler))
	defer ts.Close()

	resp, err := ts.Client().Get(ts.URL)
	if err != nil {
		t.Fatalf("hold request failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if got := s.held.Count(); got != 1 {
		t.Fatalf("Count() while held = %d, want 1", got)
	}

	close(s.closing)
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()
	if err := s.held.WaitDrained(waitCtx); err != nil {
		t.Fatalf("hold was not released by server close: %v", err)
	}
}
