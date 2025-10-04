package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jpoz/taco/internal/commands"
	"github.com/jpoz/taco/internal/config"
	"github.com/jpoz/taco/internal/server"
)

func main() {
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
			fmt.Fprintf(os.Stderr, "Export error: %v\n", err)
			os.Exit(1)
		}
	case "stats":
		if err := commands.Stats(args); err != nil {
			fmt.Fprintf(os.Stderr, "Stats error: %v\n", err)
			os.Exit(1)
		}
	case "view":
		if err := commands.View(args); err != nil {
			fmt.Fprintf(os.Stderr, "View error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func startCommand(args []string) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	port := fs.Int("port", 0, "Port to listen on")
	configPath := fs.String("config", "", "Path to config file")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse flags: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if *port != 0 {
		cfg.Port = *port
	}

	srv := server.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Taco - LLM API Proxy")
	fmt.Println("\nUsage:")
	fmt.Println("  taco start [--port 4567] [--config ./config.json]")
	fmt.Println("  taco export [--from YYYY-MM-DD] [--to YYYY-MM-DD] [--provider claude|openai] [--output file.jsonl]")
	fmt.Println("  taco stats [--from YYYY-MM-DD] [--provider claude|openai]")
	fmt.Println("  taco view <recording-id>")
	fmt.Println("  taco help")
	fmt.Println("\nCommands:")
	fmt.Println("  start   - Start the proxy server")
	fmt.Println("  export  - Export recordings to a file")
	fmt.Println("  stats   - Show statistics about recordings")
	fmt.Println("  view    - View a specific recording")
	fmt.Println("  help    - Show this help message")
}
