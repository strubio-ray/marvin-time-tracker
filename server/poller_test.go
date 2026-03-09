package main

import (
	"testing"
	"time"
)

type mockMarvinClient struct {
	trackedItem    *TrackedItemResponse
	trackCalls     []struct{ TaskID, Action string }
	retrackCalls   []struct{ TaskID string; Times []int64 }
	updateDocCalls []struct{ TaskID string; Setters []DocSetter }
}

func (m *mockMarvinClient) GetTrackedItem() (*TrackedItemResponse, error) {
	return m.trackedItem, nil
}

func (m *mockMarvinClient) Track(taskID, action string) error {
	m.trackCalls = append(m.trackCalls, struct{ TaskID, Action string }{taskID, action})
	return nil
}

func (m *mockMarvinClient) Retrack(taskID string, times []int64) error {
	m.retrackCalls = append(m.retrackCalls, struct{ TaskID string; Times []int64 }{taskID, times})
	return nil
}

func (m *mockMarvinClient) UpdateDoc(taskID string, setters []DocSetter) error {
	m.updateDocCalls = append(m.updateDocCalls, struct{ TaskID string; Setters []DocSetter }{taskID, setters})
	return nil
}

type mockNotifier struct {
	startCalls      int
	updateCalls     int
	endCalls        int
	silentPushCalls int
	alertPushCalls  int

	lastSilentToken string
	lastAlertToken  string
	lastAlertTitle  string
	lastAlertBody   string
}

func (m *mockNotifier) StartActivity(token, title string, startedAt int64) error {
	m.startCalls++
	return nil
}
func (m *mockNotifier) UpdateActivity(token, title string, startedAt int64) error {
	m.updateCalls++
	return nil
}
func (m *mockNotifier) EndActivity(token string) error {
	m.endCalls++
	return nil
}
func (m *mockNotifier) SendSilentPush(deviceToken string, taskTitle string, startedAtMs int64) error {
	m.silentPushCalls++
	m.lastSilentToken = deviceToken
	return nil
}
func (m *mockNotifier) SendAlertPush(deviceToken string, title string, body string) error {
	m.alertPushCalls++
	m.lastAlertToken = deviceToken
	m.lastAlertTitle = title
	m.lastAlertBody = body
	return nil
}

func TestPollerDetectsMissedStart(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.PushToStartToken = "token-123"
	})

	mc := &mockMarvinClient{
		trackedItem: &TrackedItemResponse{
			TaskID:    "task-1",
			Title:     "Missed Task",
			StartedAt: time.Now().UnixMilli(),
		},
	}
	notifier := &mockNotifier{}

	poller := NewPoller(mc, store, notifier, 30*time.Second, 5*time.Minute, nil)
	poller.poll()

	state := store.Get()
	if !state.IsTracking() {
		t.Error("expected tracking after missed start detection")
	}
	if state.TrackingTaskID != "task-1" {
		t.Errorf("expected task-1, got %s", state.TrackingTaskID)
	}
	if notifier.startCalls != 1 {
		t.Errorf("expected 1 start call, got %d", notifier.startCalls)
	}
}

func TestPollerDetectsMissedStop(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Running Task"
		s.StartedAt = time.Now().UnixMilli()
		s.UpdateToken = "update-token"
	})

	mc := &mockMarvinClient{trackedItem: nil}
	notifier := &mockNotifier{}

	poller := NewPoller(mc, store, notifier, 30*time.Second, 5*time.Minute, nil)
	poller.poll()

	state := store.Get()
	if state.IsTracking() {
		t.Error("expected not tracking after missed stop detection")
	}
	if notifier.endCalls != 1 {
		t.Errorf("expected 1 end call, got %d", notifier.endCalls)
	}
}

func TestPollerIgnoresStaleTrackedItem(t *testing.T) {
	now := time.Now()
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.LastStopAt = now
		s.PushToStartToken = "token-123"
	})

	mc := &mockMarvinClient{
		trackedItem: &TrackedItemResponse{
			TaskID:    "task-1",
			Title:     "Stale Task",
			StartedAt: now.Add(-1 * time.Second).UnixMilli(), // Started BEFORE stop
		},
	}
	notifier := &mockNotifier{}

	poller := NewPoller(mc, store, notifier, 30*time.Second, 5*time.Minute, nil)
	poller.poll()

	state := store.Get()
	if state.IsTracking() {
		t.Error("expected poller to ignore stale tracked item (startedAt < LastStopAt)")
	}
	if notifier.startCalls != 0 {
		t.Errorf("expected 0 start calls, got %d", notifier.startCalls)
	}
}

func TestPollerAcceptsNewTrackedItem(t *testing.T) {
	now := time.Now()
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.LastStopAt = now.Add(-10 * time.Second)
		s.PushToStartToken = "token-123"
	})

	mc := &mockMarvinClient{
		trackedItem: &TrackedItemResponse{
			TaskID:    "task-1",
			Title:     "New Task",
			StartedAt: now.Add(-5 * time.Second).UnixMilli(), // Started AFTER stop
		},
	}
	notifier := &mockNotifier{}

	poller := NewPoller(mc, store, notifier, 30*time.Second, 5*time.Minute, nil)
	poller.poll()

	state := store.Get()
	if !state.IsTracking() {
		t.Error("expected poller to accept new tracked item (startedAt >= LastStopAt)")
	}
}

func TestPollerIntervalAdaptation(t *testing.T) {
	poller := &Poller{
		activeInterval: 30 * time.Second,
		idleInterval:   5 * time.Minute,
	}

	if poller.currentInterval(true) != 30*time.Second {
		t.Error("expected active interval when tracking")
	}
	if poller.currentInterval(false) != 5*time.Minute {
		t.Error("expected idle interval when not tracking")
	}
}
