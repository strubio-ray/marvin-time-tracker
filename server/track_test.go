package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockMarvinClient struct {
	trackCalls       []struct{ TaskID, Action string }
	retrackCalls     []struct{ TaskID string; Times []int64 }
	updateDocCalls   []struct{ TaskID string; Setters []DocSetter }
	allCalls         []string
	todayItemsResult []byte
	todayItemsErr    bool
	todayItemsCalls  int
}

func (m *mockMarvinClient) Track(taskID, action string) error {
	m.trackCalls = append(m.trackCalls, struct{ TaskID, Action string }{taskID, action})
	m.allCalls = append(m.allCalls, "track:"+action)
	return nil
}

func (m *mockMarvinClient) Retrack(taskID string, times []int64) error {
	m.retrackCalls = append(m.retrackCalls, struct{ TaskID string; Times []int64 }{taskID, times})
	m.allCalls = append(m.allCalls, "retrack")
	return nil
}

func (m *mockMarvinClient) UpdateDoc(taskID string, setters []DocSetter) error {
	m.updateDocCalls = append(m.updateDocCalls, struct{ TaskID string; Setters []DocSetter }{taskID, setters})
	m.allCalls = append(m.allCalls, "updateDoc")
	return nil
}

func (m *mockMarvinClient) TodayItems() ([]byte, error) {
	m.todayItemsCalls++
	if m.todayItemsErr {
		return nil, fmt.Errorf("marvin error")
	}
	return m.todayItemsResult, nil
}

func TestTrackHandlerStart(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.PushToStartToken = "pts-token"
	})
	mc := &mockMarvinClient{}
	notifier := &mockNotifier{}
	th := NewTrackHandler(store, mc, notifier, nil, nil)

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
	th := NewTrackHandler(store, mc, notifier, nil, nil)

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
	th := NewTrackHandler(store, mc, nil, nil, nil)

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
	th := NewTrackHandler(store, mc, nil, nil, nil)

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
	th := NewTrackHandler(store, mc, nil, nil, nil)

	body, _ := json.Marshal(stopRequest{})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTrackHandlerStopAPICallOrder(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Running"
		s.StartedAt = 12345
		s.Times = []int64{100, 200, 300}
	})
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil, nil, nil)

	body, _ := json.Marshal(stopRequest{})
	req := httptest.NewRequest(http.MethodPost, "/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify Retrack is called before Track(STOP)
	if len(mc.allCalls) < 2 {
		t.Fatalf("expected at least 2 API calls, got %d", len(mc.allCalls))
	}
	if mc.allCalls[0] != "retrack" {
		t.Errorf("expected first call to be retrack, got %s", mc.allCalls[0])
	}
	if mc.allCalls[1] != "track:STOP" {
		t.Errorf("expected second call to be track:STOP, got %s", mc.allCalls[1])
	}
}

func TestTrackHandlerStartMissingTaskID(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	mc := &mockMarvinClient{}
	th := NewTrackHandler(store, mc, nil, nil, nil)

	body, _ := json.Marshal(startRequest{Title: "No ID"})
	req := httptest.NewRequest(http.MethodPost, "/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	th.HandleStart(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
