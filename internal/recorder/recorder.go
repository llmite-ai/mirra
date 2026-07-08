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
	ID           string       `json:"id"`
	Timestamp    time.Time    `json:"timestamp"`
	Provider     string       `json:"provider"`
	Request      RequestData  `json:"request"`
	Response     ResponseData `json:"response"`
	ResponseSize int64        `json:"responseSize"`
	Timing       TimingData   `json:"timing"`
	Error        string       `json:"error,omitempty"`
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
	enabled      bool
	path         string
	mu           sync.Mutex
	recordChan   chan Recording
	stopChan     chan struct{}
	wg           sync.WaitGroup
	index        *Index
	inflight     *InflightTracker
	groupManager GroupManager
}

// GroupManager is an interface for grouping recordings
// This allows the recorder to be decoupled from the grouping implementation
type GroupManager interface {
	OnRecordingWrite(*Recording) error
	Close() error
}

func New(enabled bool, path string) *Recorder {
	r := &Recorder{
		enabled:    enabled,
		path:       path,
		recordChan: make(chan Recording, 100),
		stopChan:   make(chan struct{}),
		index:      NewIndex(path),
		inflight:   NewInflightTracker(),
	}

	if enabled {
		if err := os.MkdirAll(path, 0755); err != nil {
			slog.Error("failed to create recordings directory", "error", err, "path", path)
			r.enabled = false
			return r
		}

		// Load or rebuild index
		if err := r.index.Load(); err != nil {
			slog.Error("failed to load index, will rebuild", "error", err)
			if err := r.index.Rebuild(); err != nil {
				slog.Error("failed to rebuild index", "error", err)
			}
		} else if r.index.Size() == 0 {
			// Index is empty, rebuild it
			slog.Info("index is empty, rebuilding from existing recordings")
			if err := r.index.Rebuild(); err != nil {
				slog.Error("failed to rebuild index", "error", err)
			}
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

	// Get current file size to determine offset
	stat, err := os.Stat(fullPath)
	var offset int64
	if err == nil {
		offset = stat.Size()
	}

	f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal recording: %w", err)
	}

	length := int64(len(data))
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write recording: %w", err)
	}

	// Update index
	r.index.Add(IndexEntry{
		ID:        rec.ID,
		Filename:  filename,
		Offset:    offset,
		Length:    length,
		Timestamp: rec.Timestamp,
		Provider:  rec.Provider,
	})

	// Update grouping indexes if enabled
	if r.groupManager != nil {
		if err := r.groupManager.OnRecordingWrite(&rec); err != nil {
			slog.Error("failed to update grouping indexes", "error", err, "id", rec.ID)
			// Don't fail the recording write if grouping fails
		}
	}

	return nil
}

func (r *Recorder) Close() error {
	if !r.enabled {
		return nil
	}

	close(r.stopChan)
	r.wg.Wait()

	// Close grouping manager first
	if r.groupManager != nil {
		if err := r.groupManager.Close(); err != nil {
			slog.Error("failed to close grouping manager", "error", err)
		}
	}

	// Save index before closing
	if err := r.index.Save(); err != nil {
		slog.Error("failed to save index", "error", err)
		return err
	}

	return nil
}

// SetGroupManager sets the group manager for this recorder
func (r *Recorder) SetGroupManager(gm GroupManager) {
	r.groupManager = gm
}

// GetIndex returns the recorder's index for use by API handlers
func (r *Recorder) GetIndex() *Index {
	return r.index
}

// TrackStart registers a request as in-flight. Tracking is independent of
// whether recording-to-disk is enabled.
func (r *Recorder) TrackStart(rec *Recording) {
	r.inflight.Start(rec)
}

// TrackDone removes a request from the in-flight set once it has completed.
func (r *Recorder) TrackDone(id string) {
	r.inflight.Done(id)
}

// InflightRequests returns the requests currently being proxied, newest first.
func (r *Recorder) InflightRequests() []InflightRequest {
	return r.inflight.List()
}

func NewRecording(provider, method, path, query string, startTime time.Time) Recording {
	// Generate timestamp-prefixed ID for efficient file-based lookups
	// Format: YYYYMMDD-{uuid} allows us to determine which day's file to search
	timestamp := time.Now()
	datePrefix := timestamp.Format("20060102")
	id := fmt.Sprintf("%s-%s", datePrefix, uuid.New().String())

	return Recording{
		ID:        id,
		Timestamp: timestamp,
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
