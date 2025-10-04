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

func Stats(args []string) error {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	from := fs.String("from", "", "Start date (YYYY-MM-DD)")
	provider := fs.String("provider", "", "Filter by provider (claude|openai)")
	recordingsPath := fs.String("recordings", "./recordings", "Path to recordings directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	var fromDate time.Time
	var err error

	if *from != "" {
		fromDate, err = time.Parse("2006-01-02", *from)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
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

	stats := &Statistics{
		ByProvider: make(map[string]*ProviderStats),
	}

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

		// Check if file is after from date
		if *from != "" && fileDate.Before(fromDate) {
			continue
		}

		// Read and process recordings
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

			// Apply filters
			if *provider != "" && rec.Provider != *provider {
				continue
			}

			if *from != "" && rec.Timestamp.Before(fromDate) {
				continue
			}

			stats.addRecording(&rec)
		}

		f.Close()
	}

	stats.print()
	return nil
}

type Statistics struct {
	TotalRequests int64
	TotalErrors   int64
	TotalDuration int64
	ByProvider    map[string]*ProviderStats
}

type ProviderStats struct {
	Requests int64
	Errors   int64
	Duration int64
}

func (s *Statistics) addRecording(rec *recorder.Recording) {
	s.TotalRequests++
	s.TotalDuration += rec.Timing.DurationMs

	if rec.Response.Status >= 400 {
		s.TotalErrors++
	}

	if s.ByProvider[rec.Provider] == nil {
		s.ByProvider[rec.Provider] = &ProviderStats{}
	}

	provStats := s.ByProvider[rec.Provider]
	provStats.Requests++
	provStats.Duration += rec.Timing.DurationMs

	if rec.Response.Status >= 400 {
		provStats.Errors++
	}
}

func (s *Statistics) print() {
	fmt.Println("=== Overall Statistics ===")
	fmt.Printf("Total Requests: %d\n", s.TotalRequests)
	fmt.Printf("Total Errors: %d\n", s.TotalErrors)
	if s.TotalRequests > 0 {
		fmt.Printf("Error Rate: %.2f%%\n", float64(s.TotalErrors)/float64(s.TotalRequests)*100)
		fmt.Printf("Average Response Time: %.2fms\n", float64(s.TotalDuration)/float64(s.TotalRequests))
	}

	for provider, stats := range s.ByProvider {
		fmt.Printf("\n=== %s ===\n", strings.ToUpper(provider))
		fmt.Printf("Requests: %d\n", stats.Requests)
		fmt.Printf("Errors: %d\n", stats.Errors)
		if stats.Requests > 0 {
			fmt.Printf("Error Rate: %.2f%%\n", float64(stats.Errors)/float64(stats.Requests)*100)
			fmt.Printf("Average Response Time: %.2fms\n", float64(stats.Duration)/float64(stats.Requests))
		}
	}
}
