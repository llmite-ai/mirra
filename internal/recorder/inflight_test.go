package recorder

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func mkRecording(id string, started time.Time) *Recording {
	return &Recording{
		ID:       id,
		Provider: "claude",
		Request: RequestData{
			Method: "POST",
			Path:   "/v1/messages",
		},
		Timing: TimingData{StartedAt: started},
	}
}

func TestInflightTracker_StartCopiesFields(t *testing.T) {
	tracker := NewInflightTracker()
	started := time.Now()
	tracker.Start(mkRecording("20260708-abc", started))

	list := tracker.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 in-flight request, got %d", len(list))
	}

	got := list[0]
	if got.ID != "20260708-abc" || got.Provider != "claude" ||
		got.Method != "POST" || got.Path != "/v1/messages" || !got.StartedAt.Equal(started) {
		t.Fatalf("in-flight request did not capture recording fields: %+v", got)
	}
}

func TestInflightTracker_StartNilIsNoop(t *testing.T) {
	tracker := NewInflightTracker()
	tracker.Start(nil)

	if len(tracker.List()) != 0 {
		t.Fatalf("expected nil Start to be a no-op")
	}
}

func TestInflightTracker_DoneRemoves(t *testing.T) {
	tracker := NewInflightTracker()
	tracker.Start(mkRecording("a", time.Now()))
	tracker.Start(mkRecording("b", time.Now()))

	tracker.Done("a")

	list := tracker.List()
	if len(list) != 1 || list[0].ID != "b" {
		t.Fatalf("expected only b to remain, got %+v", list)
	}
}

func TestInflightTracker_DoneUnknownIsNoop(t *testing.T) {
	tracker := NewInflightTracker()
	tracker.Start(mkRecording("a", time.Now()))

	tracker.Done("does-not-exist")

	if len(tracker.List()) != 1 {
		t.Fatalf("removing an unknown id should not affect the set")
	}
}

func TestInflightTracker_ListNewestFirst(t *testing.T) {
	tracker := NewInflightTracker()
	base := time.Now()
	// Insert out of chronological order to prove List sorts, not insertion order.
	tracker.Start(mkRecording("old", base.Add(-2*time.Second)))
	tracker.Start(mkRecording("new", base))
	tracker.Start(mkRecording("mid", base.Add(-1*time.Second)))

	list := tracker.List()
	want := []string{"new", "mid", "old"}
	if len(list) != len(want) {
		t.Fatalf("expected %d requests, got %d", len(want), len(list))
	}
	for i, id := range want {
		if list[i].ID != id {
			t.Fatalf("expected newest-first order %v, got index %d = %s", want, i, list[i].ID)
		}
	}
}

func TestInflightTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewInflightTracker()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("req-%d", n)
			tracker.Start(mkRecording(id, time.Now()))
			_ = tracker.List()
			tracker.Done(id)
		}(i)
	}

	wg.Wait()

	if len(tracker.List()) != 0 {
		t.Fatalf("expected all requests removed, got %d", len(tracker.List()))
	}
}
