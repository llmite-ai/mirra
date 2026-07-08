package recorder

import (
	"sort"
	"sync"
	"time"
)

// InflightRequest is the live view of a request currently being proxied,
// captured at request start before any response exists.
type InflightRequest struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	StartedAt time.Time `json:"started_at"`
}

// InflightTracker holds the set of requests the proxy is currently handling.
// It is independent of recording-to-disk so the UI can show live traffic even
// when persistence is disabled.
type InflightTracker struct {
	mu       sync.RWMutex
	requests map[string]InflightRequest
}

// NewInflightTracker creates an empty tracker.
func NewInflightTracker() *InflightTracker {
	return &InflightTracker{
		requests: make(map[string]InflightRequest),
	}
}

// Start registers a recording as in-flight. Only the fields known at request
// start are copied; the recording itself keeps mutating as the response streams.
func (t *InflightTracker) Start(rec *Recording) {
	if rec == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.requests[rec.ID] = InflightRequest{
		ID:        rec.ID,
		Provider:  rec.Provider,
		Method:    rec.Request.Method,
		Path:      rec.Request.Path,
		StartedAt: rec.Timing.StartedAt,
	}
}

// Done removes a request from the in-flight set. Unknown IDs are ignored.
func (t *InflightTracker) Done(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.requests, id)
}

// List returns the current in-flight requests, newest first.
func (t *InflightTracker) List() []InflightRequest {
	t.mu.RLock()
	defer t.mu.RUnlock()

	requests := make([]InflightRequest, 0, len(t.requests))
	for _, req := range t.requests {
		requests = append(requests, req)
	}

	sort.Slice(requests, func(i, j int) bool {
		return requests[i].StartedAt.After(requests[j].StartedAt)
	})

	return requests
}
