package main

import (
	"path/filepath"
	"sync"
	"testing"
)

func tempHistoryFile(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "history.json")
}

func TestHistoryStoreRoundTrip(t *testing.T) {
	path := tempHistoryFile(t)
	hs := NewHistoryStore(path)

	if err := hs.Add(SessionRecord{
		TaskID:    "task-1",
		Title:     "First Task",
		StartedAt: 1000,
		StoppedAt: 2000,
		Duration:  1000,
	}); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	if err := hs.Add(SessionRecord{
		TaskID:    "task-2",
		Title:     "Second Task",
		StartedAt: 3000,
		StoppedAt: 4000,
		Duration:  1000,
	}); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	// Reload from disk
	hs2 := NewHistoryStore(path)
	if err := hs2.Load(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	sessions := hs2.Recent(10)
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].TaskID != "task-2" {
		t.Errorf("expected newest first (task-2), got %s", sessions[0].TaskID)
	}
	if sessions[1].TaskID != "task-1" {
		t.Errorf("expected task-1 second, got %s", sessions[1].TaskID)
	}
}

func TestHistoryStoreLoadMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	hs := NewHistoryStore(path)

	if err := hs.Load(); err != nil {
		t.Fatalf("load should not fail for missing file: %v", err)
	}

	sessions := hs.Recent(10)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestHistoryStorePruning(t *testing.T) {
	path := tempHistoryFile(t)
	hs := NewHistoryStore(path)

	for i := 0; i < 205; i++ {
		if err := hs.Add(SessionRecord{
			TaskID:    "task",
			StartedAt: int64(i),
			StoppedAt: int64(i + 1),
			Duration:  1,
		}); err != nil {
			t.Fatalf("add %d failed: %v", i, err)
		}
	}

	sessions := hs.Recent(300)
	if len(sessions) != 200 {
		t.Errorf("expected 200 sessions after pruning, got %d", len(sessions))
	}

	// Newest should be the last added (StartedAt=204)
	if sessions[0].StartedAt != 204 {
		t.Errorf("expected newest StartedAt=204, got %d", sessions[0].StartedAt)
	}
}

func TestHistoryStoreRecent(t *testing.T) {
	path := tempHistoryFile(t)
	hs := NewHistoryStore(path)

	for i := 0; i < 5; i++ {
		hs.Add(SessionRecord{
			TaskID:    "task",
			Title:     "Task",
			StartedAt: int64(i * 1000),
			StoppedAt: int64(i*1000 + 500),
			Duration:  500,
		})
	}

	// Request fewer than available
	sessions := hs.Recent(3)
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}
	// Newest first (last added = StartedAt 4000)
	if sessions[0].StartedAt != 4000 {
		t.Errorf("expected StartedAt=4000, got %d", sessions[0].StartedAt)
	}

	// Request more than available
	sessions = hs.Recent(100)
	if len(sessions) != 5 {
		t.Errorf("expected 5 sessions (all available), got %d", len(sessions))
	}

	// Request zero
	sessions = hs.Recent(0)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestHistoryStoreConcurrentAccess(t *testing.T) {
	path := tempHistoryFile(t)
	hs := NewHistoryStore(path)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			hs.Add(SessionRecord{
				TaskID:    "task",
				StartedAt: int64(n),
				StoppedAt: int64(n + 1),
				Duration:  1,
			})
			_ = hs.Recent(5)
		}(i)
	}
	wg.Wait()

	// If we got here without a race/panic, the mutex is working
	sessions := hs.Recent(200)
	if len(sessions) != 50 {
		t.Errorf("expected 50 sessions, got %d", len(sessions))
	}
}
