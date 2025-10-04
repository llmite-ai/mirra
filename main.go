package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jpoz/taco/internal/commands"
	"github.com/jpoz/taco/internal/config"
	"github.com/jpoz/taco/internal/logger"
	"github.com/jpoz/taco/internal/server"
)

func main() {
	// Initialize default logger for commands
	log := logger.NewLogger("pretty", "info", os.Stdout)
	slog.SetDefault(log)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "start":
		startCommand(args)
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

	srv := server.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("shutting down")
		cancel()
	}()

	if err := srv.Start(ctx); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func printUsage() {
	usage := `Taco - LLM API Proxy

Usage:
  taco start [--port 4567] [--config ./config.json]
  taco export [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--provider claude|openai] [--output file.jsonl]
  taco stats [--from YYYY-MM-DD] [--provider claude|openai]
  taco view <recording-id>
  taco help

Commands:
  start   - Start the proxy server
  export  - Export recordings to a file
  stats   - Show statistics about recordings
  view    - View a specific recording
  help    - Show this help message`
	fmt.Fprintln(os.Stdout, usage)
}
