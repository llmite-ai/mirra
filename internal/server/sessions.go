package server

import (
	"context"
	"net/http"
	"sync"
)

// HeaderMirra marks HTTP responses as coming from a mirra server, so another
// mirra process can tell a live proxy apart from an unrelated service that
// happens to answer on the same port.
const HeaderMirra = "X-Mirra"

// sessionTracker counts CLI sessions attached to this server from other
// mirra processes (`mirra claude` reusing a running proxy), so the owning
// process can delay shutdown until they finish.
type sessionTracker struct {
	mu      sync.Mutex
	count   int
	changed chan struct{}
}

func newSessionTracker() *sessionTracker {
	return &sessionTracker{changed: make(chan struct{})}
}

func (t *sessionTracker) add(delta int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.count += delta
	close(t.changed)
	t.changed = make(chan struct{})
}

func (t *sessionTracker) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.count
}

// WaitDrained blocks until no sessions are held or ctx is done.
func (t *sessionTracker) WaitDrained(ctx context.Context) error {
	for {
		t.mu.Lock()
		if t.count == 0 {
			t.mu.Unlock()
			return nil
		}
		changed := t.changed
		t.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// HeldSessions reports how many external CLI sessions are attached.
func (s *Server) HeldSessions() int {
	return s.held.Count()
}

// WaitHeldSessionsDrained blocks until every external session detaches or
// ctx is done.
func (s *Server) WaitHeldSessionsDrained(ctx context.Context) error {
	return s.held.WaitDrained(ctx)
}

// holdHandler counts the client as an attached session for as long as it
// keeps this request open. Clients hold the connection rather than sending
// heartbeats so that process death — clean or not — releases the hold.
func (s *Server) holdHandler(w http.ResponseWriter, r *http.Request) {
	s.held.add(1)
	defer s.held.add(-1)

	w.Header().Set(HeaderMirra, "1")
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	s.log.Info("session attached", "remote", r.RemoteAddr)
	select {
	case <-r.Context().Done():
	case <-s.closing:
	}
	s.log.Info("session detached", "remote", r.RemoteAddr)
}
