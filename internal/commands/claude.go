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
// proxy. A mirra already listening on the configured port is reused;
// otherwise one is started in-process and kept alive until this session and
// any other attached sessions finish. The proxy address is injected only
// into the child's environment, so no config files are touched and nothing
// needs restoring afterwards. All arguments pass through to claude.
func Claude(args []string) (int, error) {
	cfg, err := config.Load("")
	if err != nil {
		return 1, fmt.Errorf("load config: %w", err)
	}
	proxyURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
	pretty := cfg.Logging.Format != "json" && cfg.Logging.Format != "plain"

	if mirraAlive(proxyURL) {
		return runAttached(proxyURL, args, pretty)
	}
	return runOwned(cfg, proxyURL, args, pretty)
}

// runAttached rides on a mirra proxy owned by another process.
func runAttached(proxyURL string, args []string, pretty bool) (int, error) {
	release, err := holdSession(proxyURL)
	if err != nil {
		// Registration is best-effort: an older mirra without the hold
		// endpoint still proxies fine, it just can't wait for this session
		// before shutting down.
		slog.Warn("could not register session with the running mirra", "error", err)
	} else {
		defer release()
	}
	announce(pretty, fmt.Sprintf("using the mirra already running at %s", proxyURL))
	return runClaude(proxyURL, args)
}

// runOwned starts a proxy in-process, runs claude against it, then keeps the
// proxy alive until every other attached session has finished.
func runOwned(cfg *config.Config, proxyURL string, args []string, pretty bool) (int, error) {
	// Server request logs must not scribble over claude's TUI, so they go to
	// a file for the life of the session. The startup banner still prints to
	// stdout before claude takes over the terminal.
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

	srv := server.New(cfg, log, ui.NewManager(ui.WithLogger(log)))
	ready := make(chan struct{})
	srv.OnReady(func() { close(ready) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Start(ctx) }()

	select {
	case <-ready:
	case err := <-serverDone:
		// Losing the port bind usually means another mirra claimed it between
		// our health check and listen; give it a moment and ride on it.
		if waitMirraAlive(proxyURL, 5) {
			return runAttached(proxyURL, args, pretty)
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
	// A server predating the hold endpoint answers with the UI's SPA
	// fallback, which lacks the mirra header.
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
// a structured log record when pretty output is off.
func announce(pretty bool, msg string) {
	if !pretty {
		slog.Info(msg)
		return
	}
	fmt.Fprintf(os.Stdout, "  \033[1;96m◆\033[0m  %s\n", msg)
}
