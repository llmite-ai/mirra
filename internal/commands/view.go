package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/llmite-ai/taco/internal/recorder"
)

func View(args []string) error {
	fs := flag.NewFlagSet("view", flag.ExitOnError)
	recordingsPath := fs.String("recordings", "./recordings", "Path to recordings directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() < 1 {
		return fmt.Errorf("recording ID required")
	}

	recordingID := fs.Arg(0)

	// Find all recording files
	pattern := filepath.Join(*recordingsPath, "recordings-*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list recordings: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no recordings found in %s", *recordingsPath)
	}

	// Search for the recording
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var rec recorder.Recording
			if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
				continue
			}

			if rec.ID == recordingID {
				f.Close()
				printRecording(&rec)
				return nil
			}
		}

		f.Close()
	}

	return fmt.Errorf("recording not found: %s", recordingID)
}

func printRecording(rec *recorder.Recording) {
	fmt.Printf("=== Recording %s ===\n", rec.ID)
	fmt.Printf("Timestamp: %s\n", rec.Timestamp.Format(time.RFC3339))
	fmt.Printf("Provider: %s\n", rec.Provider)
	fmt.Printf("Duration: %dms\n\n", rec.Timing.DurationMs)

	fmt.Println("--- Request ---")
	fmt.Printf("Method: %s\n", rec.Request.Method)
	fmt.Printf("Path: %s\n", rec.Request.Path)

	if len(rec.Request.Headers) > 0 {
		fmt.Println("Headers:")
		for key, values := range rec.Request.Headers {
			for _, value := range values {
				// Redact authorization headers
				if key == "Authorization" || key == "X-Api-Key" {
					fmt.Printf("  %s: [REDACTED]\n", key)
				} else {
					fmt.Printf("  %s: %s\n", key, value)
				}
			}
		}
	}

	if rec.Request.Body != nil {
		fmt.Println("Body:")
		if bodyBytes, err := json.MarshalIndent(rec.Request.Body, "  ", "  "); err == nil {
			fmt.Println(string(bodyBytes))
		} else {
			fmt.Printf("%v\n", rec.Request.Body)
		}
	}

	fmt.Println("\n--- Response ---")
	fmt.Printf("Status: %d\n", rec.Response.Status)
	fmt.Printf("Streaming: %t\n", rec.Response.Streaming)

	if len(rec.Response.Headers) > 0 {
		fmt.Println("Headers:")
		for key, values := range rec.Response.Headers {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}

	if rec.Response.Body != nil {
		fmt.Println("Body:")
		if bodyBytes, err := json.MarshalIndent(rec.Response.Body, "  ", "  "); err == nil {
			fmt.Println(string(bodyBytes))
		} else {
			fmt.Printf("%v\n", rec.Response.Body)
		}
	}
}
