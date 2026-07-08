package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jpoz/mirra/internal/attach"
	"github.com/jpoz/mirra/internal/commands"
	"github.com/jpoz/mirra/internal/config"
	"github.com/jpoz/mirra/internal/logger"
	"github.com/jpoz/mirra/internal/server"
	"github.com/jpoz/mirra/internal/ui"
)

func main() {
	// Initialize default logger for commands
	log := logger.NewLogger(os.Getenv("LOG_OUTPUT"), os.Getenv("LOG_LEVEL"), os.Stdout)
	slog.SetDefault(log)

	if len(os.Args) < 2 {
		// Bare `mirra` is shorthand for `mirra start`.
		startCommand(nil)
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	// `mirra --port 4567` and friends run as `mirra start` with those flags.
	if command != "-h" && command != "--help" && len(command) > 0 && command[0] == '-' {
		startCommand(os.Args[1:])
		return
	}

	switch command {
	case "start":
		startCommand(args)
	case "claude":
		code, err := commands.Claude(args)
		if err != nil {
			// Not slog: in owned-proxy mode the default logger points at
			// ~/.mirra/mirra.log by now, and this must reach the terminal.
			fmt.Fprintf(os.Stderr, "mirra claude: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
		if code != 0 {
			os.Exit(code)
		}
	case "export":
		if err := commands.Export(args); err != nil {
			slog.Error("export failed", "error", err)
			os.Exit(1)
		}
	case "stats":
		if err := commands.Stats(args); err != nil {
			slog.Error("stats failed", "error", err)
			os.Exit(1)
		}
	case "view":
		if err := commands.View(args); err != nil {
			slog.Error("view failed", "error", err)
			os.Exit(1)
		}
	case "reindex":
		if err := commands.Reindex(args); err != nil {
			slog.Error("reindex failed", "error", err)
			os.Exit(1)
		}
	case "groups":
		if err := commands.Groups(args); err != nil {
			slog.Error("groups failed", "error", err)
			os.Exit(1)
		}
	case "clear":
		if err := commands.Clear(args); err != nil {
			slog.Error("clear failed", "error", err)
			os.Exit(1)
		}
	case "detach":
		if err := commands.Detach(args); err != nil {
			slog.Error("detach failed", "error", err)
			os.Exit(1)
		}
	case "open":
		if err := commands.Open(args); err != nil {
			slog.Error("open failed", "error", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		slog.Error("unknown command", "command", command)
		printUsage()
		os.Exit(1)
	}
}

func startCommand(args []string) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.Int("port", 0, "Port to listen on")
	configPath := fs.String("config", "", "Path to config file")
	attachTargets := fs.String("attach", "", "Comma-separated CLIs whose new sessions should use the proxy (claude,codex)")

	if err := fs.Parse(args); err != nil {
		slog.Error("failed to parse flags", "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if *port != 0 {
		cfg.Port = *port
	}

	// Reinitialize logger with config settings
	log := logger.NewLogger(cfg.Logging.Format, cfg.Logging.Level, os.Stdout)
	slog.SetDefault(log)

	var attachMgr *attach.Manager
	if *attachTargets != "" {
		targets, err := attach.ParseTargets(*attachTargets)
		if err != nil {
			slog.Error("invalid --attach value", "error", err)
			os.Exit(1)
		}
		proxyURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
		attachMgr, err = attach.NewManager(targets, proxyURL, log)
		if err != nil {
			slog.Error("attach setup failed", "error", err)
			os.Exit(1)
		}
		// Same format split as the server startup banner: anything but
		// json/plain gets human-facing banner lines.
		if cfg.Logging.Format != "json" && cfg.Logging.Format != "plain" {
			attachMgr.EnablePrettyOutput()
		}
	}

	uiManager := ui.NewManager(ui.WithLogger(log))
	srv := server.New(cfg, log, uiManager)

	if attachMgr != nil {
		// Attach only once the listener is bound, so configs never point at a
		// port the proxy failed to claim.
		srv.OnReady(func() {
			if err := attachMgr.Attach(); err != nil {
				slog.Error("attach failed; proxy keeps running", "error", err)
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("shutting down")
		cancel()
	}()

	err = srv.Start(ctx)
	if attachMgr != nil {
		if derr := attachMgr.Detach(); derr != nil {
			slog.Error("detach failed; run `mirra detach` to restore configs", "error", derr)
		}
	}
	if err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func printUsage() {
	usage := `MIRRA - Monitoring & Inspection Recording Relay Archive

Usage:
  mirra                # same as: mirra start
  mirra start [--port 4567] [--config ./config.json] [--attach claude,codex]
  mirra claude [claude args...]
  mirra detach
  mirra open [--port 4567] [--config ./config.json]
  mirra export [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--provider claude|openai|gemini|chatgpt] [--output file.jsonl]
  mirra stats [--from YYYY-MM-DD] [--provider claude|openai|gemini|chatgpt]
  mirra view <recording-id>
  mirra reindex [--recordings ./recordings]
  mirra groups sessions [--limit 20] [--provider <provider>] [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--errors]
  mirra clear [--recordings ./recordings] [--force]
  mirra help

Commands:
  start    - Start the proxy server; --attach points new claude/codex sessions at it
  claude   - Run the claude CLI through mirra, reusing a running proxy or starting one
  detach   - Restore claude/codex configs if an attached run did not shut down cleanly
  open     - Open the UI of a running mirra in the browser
  export   - Export recordings to a file
  stats    - Show statistics about recordings
  view     - View a specific recording
  reindex  - Rebuild the recording index for faster lookups
  groups   - List and view session groups
  clear    - Delete all recordings and reset the database
  help     - Show this help message`
	_, _ = fmt.Fprintln(os.Stdout, usage)
}
