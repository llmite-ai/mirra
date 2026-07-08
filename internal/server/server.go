package server

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jpoz/mirra/internal/api"
	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/grouping"
	"github.com/jpoz/mirra/internal/proxy"
	"github.com/jpoz/mirra/internal/recorder"
	"github.com/jpoz/mirra/internal/ui"
)

type Server struct {
	cfg          *config.Config
	proxy        *proxy.Proxy
	recorder     *recorder.Recorder
	groupManager *grouping.Manager
	log          *slog.Logger
	uiManager    *ui.Manager
	onReady      func()
	held         *sessionTracker
	closing      chan struct{}
}

// OnReady registers a callback invoked once the listener is bound and the
// proxy is able to serve requests.
func (s *Server) OnReady(fn func()) {
	s.onReady = fn
}

func New(cfg *config.Config, log *slog.Logger, uiManager *ui.Manager) *Server {
	rec := recorder.New(cfg.Recording.Enabled, cfg.Recording.Path)

	// Initialize grouping manager if recording is enabled
	var groupMgr *grouping.Manager
	if cfg.Recording.Enabled {
		groupMgr = grouping.NewManager(cfg.Recording.Path, true)
		rec.SetGroupManager(groupMgr)
		slog.Info("grouping enabled")
	}

	return &Server{
		cfg:          cfg,
		recorder:     rec,
		groupManager: groupMgr,
		proxy:        proxy.New(cfg, rec),
		log:          log,
		uiManager:    uiManager,
		held:         newSessionTracker(),
		closing:      make(chan struct{}),
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// API handlers
	apiHandlers := api.NewHandlers(s.cfg, s.log, s.recorder)
	mux.Handle("GET /api/recordings", http.HandlerFunc(apiHandlers.ListRecordings))
	mux.Handle("GET /api/recordings/{id}/parse", http.HandlerFunc(apiHandlers.ParseRecording))
	mux.Handle("GET /api/recordings/{id}", http.HandlerFunc(apiHandlers.GetRecording))

	// Group API handlers
	if s.groupManager != nil {
		groupHandlers := api.NewGroupHandlers(s.log, s.recorder, s.groupManager)
		mux.Handle("GET /api/groups/sessions", http.HandlerFunc(groupHandlers.ListSessionGroups))
		mux.Handle("GET /api/groups/sessions/", http.HandlerFunc(groupHandlers.GetSessionGroup))
	}

	// Health check endpoint
	mux.Handle("GET /health", http.HandlerFunc(s.healthHandler))

	// Session hold: `mirra claude` riding on this proxy keeps one of these
	// requests open for the life of its CLI session so the owning process
	// knows not to shut down underneath it.
	mux.Handle("GET /api/attach/hold", http.HandlerFunc(s.holdHandler))

	// UI source files
	mux.Handle("GET /src/", s.uiManager.SrcHandler("/src"))

	// UI static files (GET requests to root). WebSocket upgrades are never
	// UI traffic — without this check the SPA fallback would answer codex's
	// upgrade with index.html; the proxy instead tunnels known API paths and
	// 404s unknown ones.
	static := s.uiManager.Static("internal/ui/static", "/")
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if proxy.IsWebSocketUpgrade(r) {
			s.proxy.Handle(w, r)
			return
		}
		static.ServeHTTP(w, r)
	}))

	// API GETs must beat the UI's "GET /" catch-all: plain API GETs like
	// /v1/models are GET requests too. Non-GET methods already reach the
	// proxy through the "/" catch-all. /v1beta, /v1alpha and /upload are
	// Gemini prefixes.
	for _, prefix := range []string{"/v1/", "/v1beta/", "/v1alpha/", "/upload/"} {
		mux.HandleFunc("GET "+prefix, s.proxy.Handle)
	}

	// Proxy catch-all (all other requests)
	mux.HandleFunc("/", s.proxy.Handle)

	// Wrap mux with logging middleware
	handler := s.loggingMiddleware(mux)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: handler,
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		if closeErr := s.recorder.Close(); closeErr != nil {
			slog.Error("recorder close error", "error", closeErr)
		}
		return err
	}

	s.printStartupBanner()

	if s.onReady != nil {
		s.onReady()
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Serve(ln)
	}()

	select {
	case err := <-errChan:
		if closeErr := s.recorder.Close(); closeErr != nil {
			slog.Error("recorder close error", "error", closeErr)
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down gracefully")
		// Release held sessions first: their handlers block until the client
		// disconnects and would otherwise pin Shutdown to its timeout.
		close(s.closing)
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

// printStartupBanner announces the server URL once the listener is bound.
// The pretty format gets a plain banner so terminals render the URL as a
// clickable link; structured formats keep a machine-parseable log line.
func (s *Server) printStartupBanner() {
	url := fmt.Sprintf("http://localhost:%d", s.cfg.Port)

	switch s.cfg.Logging.Format {
	case "json", "plain":
		slog.Info("𝕄𝕀ℝℝ𝔸 started", "port", s.cfg.Port, "url", url)
	default:
		fmt.Fprintf(os.Stdout, "\n  𝕄𝕀ℝℝ𝔸 running\n\n  ➜  UI & proxy: \033[4;96m%s\033[0m\n\n", url)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	// The header identifies this as a mirra proxy so discovery (`mirra
	// claude` looking for a server to reuse) can't be fooled by an unrelated
	// service on the same port.
	w.Header().Set(HeaderMirra, "1")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"service":"mirra","status":"ok"}`))
}

// GetRecorder returns the server's recorder instance
func (s *Server) GetRecorder() *recorder.Recorder {
	return s.recorder
}

// loggingMiddleware wraps an http.Handler to log request details
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(ww, r)

		duration := time.Since(start)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.statusCode,
			"duration", duration,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack exposes the underlying connection's Hijacker so the proxy can take
// over the TCP stream for WebSocket tunneling; the middleware wrapper would
// otherwise hide it. A hijacked connection means a completed 101 upgrade.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	rw.statusCode = http.StatusSwitchingProtocols
	return hijacker.Hijack()
}
