package recorder

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Recording struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Provider  string                 `json:"provider"`
	Request   RequestData            `json:"request"`
	Response  ResponseData           `json:"response"`
	Timing    TimingData             `json:"timing"`
	Error     string                 `json:"error,omitempty"`
}

type RequestData struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Query   string              `json:"query,omitempty"`
	Headers map[string][]string `json:"headers"`
	Body    interface{}         `json:"body,omitempty"`
}

type ResponseData struct {
	Status    int                 `json:"status"`
	Headers   map[string][]string `json:"headers"`
	Body      interface{}         `json:"body,omitempty"`
	Streaming bool                `json:"streaming"`
}

type TimingData struct {
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	DurationMs  int64     `json:"duration_ms"`
}

type Recorder struct {
	enabled    bool
	path       string
	mu         sync.Mutex
	recordChan chan Recording
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

func New(enabled bool, path string) *Recorder {
	r := &Recorder{
		enabled:    enabled,
		path:       path,
		recordChan: make(chan Recording, 100),
		stopChan:   make(chan struct{}),
	}

	if enabled {
		if err := os.MkdirAll(path, 0755); err != nil {
			slog.Error("failed to create recordings directory", "error", err, "path", path)
			r.enabled = false
			return r
		}

		r.wg.Add(1)
		go r.worker()
	}

	return r
}

func (r *Recorder) Record(rec Recording) {
	if !r.enabled {
		return
	}

	select {
	case r.recordChan <- rec:
	default:
		slog.Warn("recording channel full, dropping recording", "id", rec.ID)
	}
}

func (r *Recorder) worker() {
	defer r.wg.Done()

	for {
		select {
		case rec := <-r.recordChan:
			if err := r.writeRecording(rec); err != nil {
				slog.Error("failed to write recording", "error", err, "id", rec.ID)
			}
		case <-r.stopChan:
			// Drain remaining recordings
			for {
				select {
				case rec := <-r.recordChan:
					if err := r.writeRecording(rec); err != nil {
						slog.Error("failed to write recording", "error", err, "id", rec.ID)
					}
				default:
					return
				}
			}
		}
	}
}

func (r *Recorder) writeRecording(rec Recording) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	filename := fmt.Sprintf("recordings-%s.jsonl", time.Now().Format("2006-01-02"))
	fullPath := filepath.Join(r.path, filename)

	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal recording: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write recording: %w", err)
	}

	return nil
}

func (r *Recorder) Close() error {
	if !r.enabled {
		return nil
	}

	close(r.stopChan)
	r.wg.Wait()
	return nil
}

func NewRecording(provider, method, path, query string, startTime time.Time) Recording {
	return Recording{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Provider:  provider,
		Request: RequestData{
			Method:  method,
			Path:    path,
			Query:   query,
			Headers: make(map[string][]string),
		},
		Response: ResponseData{
			Headers: make(map[string][]string),
		},
		Timing: TimingData{
			StartedAt: startTime,
		},
	}
}
