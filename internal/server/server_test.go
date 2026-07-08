package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/ui"
)

func TestServer_EphemeralPort(t *testing.T) {
	cfg, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Port = 0
	cfg.Recording.Enabled = false
	cfg.Logging.Format = "json"

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, log, ui.NewManager(ui.WithLogger(log)))
	ready := make(chan struct{})
	srv.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	select {
	case <-ready:
	case err := <-done:
		t.Fatalf("server exited before ready: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server never became ready")
	}

	port := srv.Port()
	if port == 0 {
		t.Fatal("Port() = 0 after ready, want the bound ephemeral port")
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resp.Header.Get(HeaderMirra) == "" {
		t.Fatalf("health response missing %s header", HeaderMirra)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}
}
