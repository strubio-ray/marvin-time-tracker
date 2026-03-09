package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookStart(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	body, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Title:     "Test Task",
		Timestamp: 1772734813781,
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	state := store.Get()
	if state.TrackingTaskID != "task-1" {
		t.Errorf("expected task-1, got %s", state.TrackingTaskID)
	}
	if state.TaskTitle != "Test Task" {
		t.Errorf("expected Test Task, got %s", state.TaskTitle)
	}
}

func TestWebhookStop(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	// Start first
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Test Task"
		s.StartedAt = 1772734813781
	})

	body, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Timestamp: 1772734823781,
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStop(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	state := store.Get()
	if state.IsTracking() {
		t.Error("expected tracking to stop")
	}
}

func TestWebhookStartDedup(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	body, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Title:     "Test Task",
		Timestamp: 1772734813781,
	})

	// First call
	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	// Clear state to verify dedup prevents re-update
	store.Clear()

	// Second call with same dedup key
	req = httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w = httptest.NewRecorder()
	wh.HandleStart(w, req)

	state := store.Get()
	if state.IsTracking() {
		t.Error("dedup should have prevented second update")
	}
}

func TestWebhookStartInvalidJSON(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	// Should still return 200 (acknowledge-first)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if store.Get().IsTracking() {
		t.Error("invalid JSON should not start tracking")
	}
}

func TestWebhookStartIgnoresStaleTimesArray(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	// Even count = tracking stopped (stale webhook)
	body, _ := json.Marshal(map[string]interface{}{
		"_id":   "task-1",
		"title": "Stale Task",
		"times": []int64{100, 200, 300, 400},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if store.Get().IsTracking() {
		t.Error("expected stale webhook with even times count to be ignored")
	}
}

func TestWebhookStartAcceptsActiveTimesArray(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	// Odd count = tracking active
	body, _ := json.Marshal(map[string]interface{}{
		"_id":   "task-1",
		"title": "Active Task",
		"times": []int64{100, 200, 300},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	state := store.Get()
	if !state.IsTracking() {
		t.Error("expected active webhook with odd times count to start tracking")
	}
	if state.TrackingTaskID != "task-1" {
		t.Errorf("expected task-1, got %s", state.TrackingTaskID)
	}
}

func TestWebhookStartAcceptsEmptyTimesArray(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	// No times field — fail open
	body, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Title:     "No Times",
		Timestamp: time.Now().UnixMilli(),
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	if !store.Get().IsTracking() {
		t.Error("expected webhook without times array to be processed normally")
	}
}

func TestWebhookStopSetsLastStopAt(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.StartedAt = time.Now().UnixMilli()
	})

	body, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Timestamp: time.Now().UnixMilli(),
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/stop", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStop(w, req)

	state := store.Get()
	if state.LastStopAt.IsZero() {
		t.Error("expected LastStopAt to be set after webhook/stop")
	}
}

func TestWebhookStartBounceBackRejected(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.LastStoppedTaskID = "task-1"
		s.LastStopAt = time.Now()
	})
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"_id":   "task-1",
		"title": "Bounced Task",
		"times": []int64{100, 200, 300},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	if store.Get().IsTracking() {
		t.Error("expected bounce-back webhook to be rejected")
	}
}

func TestWebhookStartBounceBackDifferentTask(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.LastStoppedTaskID = "task-1"
		s.LastStopAt = time.Now()
	})
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"_id":   "task-2",
		"title": "Different Task",
		"times": []int64{100, 200, 300},
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	if !store.Get().IsTracking() {
		t.Error("expected different task to be accepted despite bounce-back guard")
	}
}

func TestWebhookStartBounceBackClearedByWebhookStop(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.StartedAt = time.Now().UnixMilli()
		s.LastStoppedTaskID = "task-1"
	})
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	// Fire webhook/stop — should clear LastStoppedTaskID
	stopBody, _ := json.Marshal(webhookPayload{
		TaskID:    "task-1",
		Timestamp: time.Now().UnixMilli(),
	})
	req := httptest.NewRequest(http.MethodPost, "/webhook/stop", bytes.NewReader(stopBody))
	w := httptest.NewRecorder()
	wh.HandleStop(w, req)

	if store.Get().LastStoppedTaskID != "" {
		t.Error("expected webhook/stop to clear LastStoppedTaskID")
	}

	// Now fire webhook/start for the same task — should be accepted
	startBody, _ := json.Marshal(map[string]interface{}{
		"_id":   "task-1",
		"title": "Restarted Task",
		"times": []int64{100, 200, 300},
	})
	req = httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(startBody))
	w = httptest.NewRecorder()
	wh.HandleStart(w, req)

	if !store.Get().IsTracking() {
		t.Error("expected webhook/start to be accepted after webhook/stop cleared bounce-back guard")
	}
}

func TestWebhookStartMissingTaskID(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	dedup := NewDedupCache(60 * time.Second)
	wh := NewWebhookHandler(store, dedup, nil, nil, nil)

	body, _ := json.Marshal(webhookPayload{Title: "No ID"})

	req := httptest.NewRequest(http.MethodPost, "/webhook/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	wh.HandleStart(w, req)

	if store.Get().IsTracking() {
		t.Error("missing taskId should not start tracking")
	}
}
