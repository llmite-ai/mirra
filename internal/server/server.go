package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jpoz/taco/internal/config"
	"github.com/jpoz/taco/internal/proxy"
	"github.com/jpoz/taco/internal/recorder"
)

type Server struct {
	cfg      *config.Config
	proxy    *proxy.Proxy
	recorder *recorder.Recorder
}

func New(cfg *config.Config) *Server {
	rec := recorder.New(cfg.Recording.Enabled, cfg.Recording.Path)

	return &Server{
		cfg:      cfg,
		recorder: rec,
		proxy:    proxy.New(cfg, rec),
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Route all /v1/* paths to the proxy
	mux.HandleFunc("/v1/", s.proxy.Handle)

	// Health check endpoint
	mux.HandleFunc("/health", s.healthHandler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("taco proxy server started",
			"port", s.cfg.Port,
			"recording", s.cfg.Recording.Enabled,
			"recordings_path", s.cfg.Recording.Path)
		errChan <- srv.ListenAndServe()
	}()

	select {
	case err := <-errChan:
		s.recorder.Close()
		return err
	case <-ctx.Done():
		slog.Info("shutting down gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}

		if err := s.recorder.Close(); err != nil {
			slog.Error("recorder close error", "error", err)
		}

		slog.Info("shutdown complete")
		return nil
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
