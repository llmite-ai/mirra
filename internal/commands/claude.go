package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/logger"
	"github.com/jpoz/mirra/internal/server"
	"github.com/jpoz/mirra/internal/ui"
)

const claudeBaseURLEnv = "ANTHROPIC_BASE_URL"

// Claude runs the claude CLI with its API traffic routed through a mirra
// proxy on the configured port (default 4567, so the UI is always at
// http://localhost:4567). A mirra already listening there is reused;
// otherwise one starts in-process and lives until this session and any other
// `mirra claude` sessions riding on it finish. The proxy address is injected
// only into the child's environment — no config files are touched, so no
// other claude session is affected. All arguments pass through to claude.
func Claude(args []string) (int, error) {
	cfg, err := config.Load("")
	if err != nil {
		return 1, fmt.Errorf("load config: %w", err)
	}
	pretty := cfg.Logging.Format != "json" && cfg.Logging.Format != "plain"

	// From here on the terminal belongs to claude's TUI, so every log —
	// including the proxy's request log — goes to a file.
	logFile, logPath, err := openRunLog()
	if err != nil {
		return 1, fmt.Errorf("open log file: %w", err)
	}
	defer func() {
		_ = logFile.Close()
	}()
	fileFormat := "plain"
	if cfg.Logging.Format == "json" {
		fileFormat = "json"
	}
	log := logger.NewLogger(fileFormat, cfg.Logging.Level, logFile)
	slog.SetDefault(log)

	proxyURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
	if mirraAlive(proxyURL) {
		if code, ok, err := attachAndRun(proxyURL, args, pretty); ok {
			return code, err
		}
	}
	return runOwned(cfg, log, logPath, args, pretty)
}

// runOwned starts a proxy in-process, runs claude against it, then keeps the
// proxy alive until every other attached `mirra claude` session has finished.
func runOwned(cfg *config.Config, log *slog.Logger, logPath string, args []string, pretty bool) (int, error) {
	srv := server.New(cfg, log, ui.NewManager(ui.WithLogger(log)))
	ready := make(chan struct{})
	srv.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Start(ctx) }()

	proxyURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
	select {
	case <-ready:
	case err := <-serverDone:
		// Losing the bind usually means another mirra claimed the port
		// between our health check and listen; give it a moment to serve.
		if waitMirraAlive(proxyURL, 5) {
			if code, ok, aerr := attachAndRun(proxyURL, args, pretty); ok {
				return code, aerr
			}
		}
		return 1, fmt.Errorf("could not start proxy on %s (another service on the port? set MIRRA_PORT to move it): %w", proxyURL, err)
	}

	announce(pretty, fmt.Sprintf("proxy logs: %s", logPath))
	code, runErr := runClaude(proxyURL, args)

	// Other `mirra claude` sessions may still be riding on this proxy; hold
	// the server open until they finish.
	if n := srv.HeldSessions(); n > 0 {
		announce(pretty, fmt.Sprintf("%d other session(s) attached; keeping proxy alive (Ctrl-C to stop now)", n))
		waitCtx, stopWait := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		err := srv.WaitHeldSessionsDrained(waitCtx)
		stopWait()
		if err == nil {
			announce(pretty, "all sessions finished")
		}
	}

	cancel()
	<-serverDone
	return code, runErr
}

// attachAndRun rides on a mirra proxy owned by another process. ok is false
// when the proxy disappeared before this session could register, in which
// case the caller should start its own.
func attachAndRun(proxyURL string, args []string, pretty bool) (code int, ok bool, err error) {
	release, err := holdSession(proxyURL)
	if err != nil {
		slog.Warn("running mirra went away; starting a new one", "url", proxyURL, "error", err)
		return 0, false, nil
	}
	defer release()
	announce(pretty, fmt.Sprintf("using the mirra already running at %s", proxyURL))
	code, err = runClaude(proxyURL, args)
	return code, true, err
}

// runClaude executes the claude CLI with ANTHROPIC_BASE_URL pointed at the
// proxy, inheriting this terminal, and returns claude's exit code.
func runClaude(proxyURL string, args []string) (int, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return 1, errors.New("claude CLI not found in PATH")
	}

	if prev := os.Getenv(claudeBaseURLEnv); prev != "" && prev != proxyURL {
		slog.Warn("overriding "+claudeBaseURLEnv+" for this session", "was", prev, "now", proxyURL)
	}

	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = claudeEnv(os.Environ(), proxyURL)

	// Ctrl-C belongs to claude: the terminal delivers SIGINT to the whole
	// foreground process group, so claude already sees it and mirra must not
	// die first. A TERM aimed at this process alone is forwarded.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start claude: %w", err)
	}
	waitDone := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				if sig == syscall.SIGTERM {
					_ = cmd.Process.Signal(sig)
				}
			case <-waitDone:
				return
			}
		}
	}()

	err = cmd.Wait()
	close(waitDone)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if code := exitErr.ExitCode(); code >= 0 {
				return code, nil
			}
			return 1, nil // killed by a signal
		}
		return 1, fmt.Errorf("run claude: %w", err)
	}
	return 0, nil
}

// claudeEnv returns base with ANTHROPIC_BASE_URL replaced by proxyURL.
func claudeEnv(base []string, proxyURL string) []string {
	env := make([]string, 0, len(base)+1)
	for _, kv := range base {
		if !strings.HasPrefix(kv, claudeBaseURLEnv+"=") {
			env = append(env, kv)
		}
	}
	return append(env, claudeBaseURLEnv+"="+proxyURL)
}

// waitMirraAlive polls for a mirra proxy at proxyURL, giving a rival process
// that just won the bind race a moment to start serving.
func waitMirraAlive(proxyURL string, attempts int) bool {
	for i := range attempts {
		if i > 0 {
			time.Sleep(200 * time.Millisecond)
		}
		if mirraAlive(proxyURL) {
			return true
		}
	}
	return false
}

// mirraAlive reports whether a mirra proxy — specifically mirra, not just
// any HTTP server — is answering at proxyURL.
func mirraAlive(proxyURL string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(proxyURL + "/health")
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return resp.StatusCode == http.StatusOK && resp.Header.Get(server.HeaderMirra) != ""
}

// holdSession opens a request the server keeps open for the life of this
// process, marking the session attached so the owning mirra waits for it
// before shutting down. The returned func releases the hold; process death
// releases it implicitly by closing the connection.
func holdSession(proxyURL string) (func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL+"/api/attach/hold", nil)
	if err != nil {
		cancel()
		return nil, err
	}
	// No client timeout: the response intentionally stays open until released.
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		cancel()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK || resp.Header.Get(server.HeaderMirra) == "" {
		cancel()
		_ = resp.Body.Close()
		return nil, fmt.Errorf("server does not support session holds (status %d)", resp.StatusCode)
	}
	return func() {
		cancel()
		_ = resp.Body.Close()
	}, nil
}

// openRunLog opens ~/.mirra/mirra.log for appending.
func openRunLog() (*os.File, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Join(home, ".mirra")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, "mirra.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, "", err
	}
	return f, path, nil
}

// announce prints a status line in the same style as the startup banner, or
// a structured log record when pretty output is off. Announcements happen
// only before claude takes over the terminal or after it exits.
func announce(pretty bool, msg string) {
	if !pretty {
		slog.Info(msg)
		return
	}
	fmt.Fprintf(os.Stdout, "  \033[1;96m◆\033[0m  %s\n", msg)
}
