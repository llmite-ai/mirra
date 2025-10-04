package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Search for the recording (supports partial UUID matching)
	var matches []recorder.Recording
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		// Increase buffer size to handle large recordings (default is 64KB)
		const maxScanTokenSize = 10 * 1024 * 1024 // 10MB
		buf := make([]byte, maxScanTokenSize)
		scanner.Buffer(buf, maxScanTokenSize)

		for scanner.Scan() {
			var rec recorder.Recording
			if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
				continue
			}

			// Support both exact match and prefix match
			if rec.ID == recordingID || len(rec.ID) >= len(recordingID) && rec.ID[:len(recordingID)] == recordingID {
				matches = append(matches, rec)
			}
		}

		f.Close()
	}

	if len(matches) == 0 {
		return fmt.Errorf("recording not found: %s", recordingID)
	}

	if len(matches) > 1 {
		fmt.Printf("error: ambiguous recording ID '%s' matches multiple recordings:\n", recordingID)
		for _, m := range matches {
			fmt.Printf("  %s\n", m.ID)
		}
		return fmt.Errorf("please provide more characters to uniquely identify the recording")
	}

	printRecording(&matches[0])
	return nil
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
		if rec.Response.Streaming {
			// Handle streaming SSE format
			if bodyStr, ok := rec.Response.Body.(string); ok {
				printSSEBody(bodyStr)
			} else {
				fmt.Printf("%v\n", rec.Response.Body)
			}
		} else {
			// Handle regular JSON responses
			if bodyBytes, err := json.MarshalIndent(rec.Response.Body, "  ", "  "); err == nil {
				fmt.Println(string(bodyBytes))
			} else {
				fmt.Printf("%v\n", rec.Response.Body)
			}
		}
	}
}

// printSSEBody parses and formats Server-Sent Events (SSE) format for better readability
func printSSEBody(body string) {
	lines := strings.Split(body, "\n")
	var currentEvent string
	var currentData strings.Builder

	for _, line := range lines {
		// SSE format: "event: event_name" and "data: json_payload"
		if strings.HasPrefix(line, "event: ") {
			// Print previous event if exists
			if currentEvent != "" && currentData.Len() > 0 {
				printSSEEvent(currentEvent, currentData.String())
				currentData.Reset()
			}
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			dataContent := strings.TrimPrefix(line, "data: ")
			if currentData.Len() > 0 {
				currentData.WriteString("\n")
			}
			currentData.WriteString(dataContent)
		} else if line == "" && currentEvent != "" && currentData.Len() > 0 {
			// Empty line signals end of event
			printSSEEvent(currentEvent, currentData.String())
			currentEvent = ""
			currentData.Reset()
		}
	}

	// Print last event if exists
	if currentEvent != "" && currentData.Len() > 0 {
		printSSEEvent(currentEvent, currentData.String())
	}
}

// printSSEEvent formats and prints a single SSE event
func printSSEEvent(eventType, data string) {
	fmt.Printf("\n  Event: %s\n", eventType)

	// Try to parse data as JSON and pretty-print it
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
		if formatted, err := json.MarshalIndent(jsonData, "    ", "  "); err == nil {
			fmt.Println(string(formatted))
			return
		}
	}

	// If not JSON or parsing failed, print raw data with indentation
	dataLines := strings.Split(data, "\n")
	for _, line := range dataLines {
		fmt.Printf("    %s\n", line)
	}
}
