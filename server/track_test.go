package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrackHandlerStart(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.PushToStartToken = "pts-token"
	})
	mc := &mockMarvinClient{}
	notifier := &mockNotifier{}
	th := NewTrackHandler(store, mc, notifier)

	body, _ := json.Marshal(startRequest{TaskID: "task-1", Title: "Test Task"})
	req := httptest.NewRequest(http.MethodPost, "/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStart(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(mc.trackCalls) != 1 || mc.trackCalls[0].Action != "START" {
		t.Errorf("expected 1 START call, got %+v", mc.trackCalls)
	}

	state := store.Get()
	if !state.IsTracking() {
		t.Error("expected tracking after start")
	}
	if notifier.startCalls != 1 {
		t.Errorf("expected 1 start notification, got %d", notifier.startCalls)
	}
}

func TestCalcDuration(t *testing.T) {
	// Two segments: (200-100) + (400-300) = 200
	times := []int64{100, 200, 300, 400}
	if d := calcDuration(times); d != 200 {
		t.Errorf("expected 200, got %d", d)
	}

	// Empty
	if d := calcDuration(nil); d != 0 {
		t.Errorf("expected 0 for nil, got %d", d)
	}

	// Odd length (unpaired start is ignored)
	times = []int64{100, 200, 300}
	if d := calcDuration(times); d != 100 {
		t.Errorf("expected 100, got %d", d)
	}
}

func TestTrackHandlerStop(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Running"
		s.StartedAt = 12345
		s.UpdateToken = "upd-token"
	})
	mc := &mockMarvinClient{}
	notifier := &mockNotifier{}
	th := NewTrackHandler(store, mc, notifier)

	body, _ := json.Marshal(stopRequest{})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(mc.trackCalls) != 1 || mc.trackCalls[0].Action != "STOP" {
		t.Errorf("expected 1 STOP call, got %+v", mc.trackCalls)
	}

	state := store.Get()
	if state.IsTracking() {
		t.Error("expected not tracking after stop")
	}
	if notifier.endCalls != 1 {
		t.Errorf("expected 1 end notification, got %d", notifier.endCalls)
	}
}

func TestTrackHandlerStopCallsRetrack(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Running"
		s.StartedAt = 12345
		s.Times = []int64{100, 200, 300}
	})
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil)

	body, _ := json.Marshal(stopRequest{})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if len(mc.retrackCalls) != 1 {
		t.Fatalf("expected 1 retrack call, got %d", len(mc.retrackCalls))
	}
	if mc.retrackCalls[0].TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", mc.retrackCalls[0].TaskID)
	}
	// Should have original 3 times + 1 stop timestamp = 4 entries (even = stopped)
	if len(mc.retrackCalls[0].Times) != 4 {
		t.Errorf("expected 4 times entries, got %d", len(mc.retrackCalls[0].Times))
	}

	state := store.Get()
	if state.Times != nil {
		t.Error("expected Times to be cleared after stop")
	}
}

func TestTrackHandlerStopWithTaskID(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
	})
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil)

	body, _ := json.Marshal(stopRequest{TaskID: "task-1"})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if mc.trackCalls[0].TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", mc.trackCalls[0].TaskID)
	}
}

func TestTrackHandlerStopNoTask(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil)

	body, _ := json.Marshal(stopRequest{})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTrackHandlerStartMissingTaskID(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil)

	body, _ := json.Marshal(startRequest{Title: "No ID"})
	req := httptest.NewRequest(http.MethodPost, "/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
