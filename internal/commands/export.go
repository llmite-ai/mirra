package commands

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/llmite-ai/taco/internal/recorder"
)

func Export(args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	from := fs.String("from", "", "Start date (YYYY-MM-DD)")
	to := fs.String("to", "", "End date (YYYY-MM-DD)")
	provider := fs.String("provider", "", "Filter by provider (claude|openai)")
	output := fs.String("output", "export.jsonl", "Output file path")
	recordingsPath := fs.String("recordings", "./recordings", "Path to recordings directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	var fromDate, toDate time.Time
	var err error

	if *from != "" {
		fromDate, err = time.Parse("2006-01-02", *from)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
	}

	if *to != "" {
		toDate, err = time.Parse("2006-01-02", *to)
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
		toDate = toDate.Add(24 * time.Hour) // Include the entire day
	} else {
		toDate = time.Now().Add(24 * time.Hour)
	}

	// Find all recording files
	pattern := filepath.Join(*recordingsPath, "recordings-*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list recordings: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no recordings found in %s", *recordingsPath)
	}

	// Create output file
	outFile, err := os.Create(*output)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	count := 0

	// Process each recording file
	for _, file := range files {
		// Extract date from filename
		base := filepath.Base(file)
		datePart := strings.TrimPrefix(base, "recordings-")
		datePart = strings.TrimSuffix(datePart, ".jsonl")

		fileDate, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}

		// Check if file is within date range
		if *from != "" && fileDate.Before(fromDate) {
			continue
		}
		if fileDate.After(toDate) {
			continue
		}

		// Read and filter recordings
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

			// Apply filters
			if *provider != "" && rec.Provider != *provider {
				continue
			}

			if *from != "" && rec.Timestamp.Before(fromDate) {
				continue
			}
			if rec.Timestamp.After(toDate) {
				continue
			}

			// Write to output
			if _, err := outFile.Write(scanner.Bytes()); err != nil {
				f.Close()
				return fmt.Errorf("failed to write to output: %w", err)
			}
			if _, err := outFile.Write([]byte("\n")); err != nil {
				f.Close()
				return fmt.Errorf("failed to write newline: %w", err)
			}

			count++
		}

		f.Close()
	}

	slog.Info("export complete", "count", count, "output", *output)
	return nil
}
