package main

import (
	"testing"
	"time"
)

func TestRenewalNotTriggeredEarly(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	now := time.Now()
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Test"
		s.StartedAt = now.UnixMilli()
		s.LiveActivityStartedAt = now
		s.UpdateToken = "update-token"
		s.PushToStartToken = "pts-token"
	})

	notifier := &mockNotifier{}
	rn := NewRenewal(store, notifier)
	rn.now = func() time.Time { return now.Add(1 * time.Hour) }
	rn.check()

	if notifier.endCalls != 0 {
		t.Errorf("expected no end calls at 1h, got %d", notifier.endCalls)
	}
}

func TestRenewalTriggeredAtThreshold(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	now := time.Now()
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Test"
		s.StartedAt = now.UnixMilli()
		s.LiveActivityStartedAt = now
		s.UpdateToken = "update-token"
		s.PushToStartToken = "pts-token"
	})

	notifier := &mockNotifier{}
	rn := NewRenewal(store, notifier)
	rn.now = func() time.Time { return now.Add(7*time.Hour + 46*time.Minute) }
	rn.check()

	if notifier.endCalls != 1 {
		t.Errorf("expected 1 end call, got %d", notifier.endCalls)
	}
	if notifier.startCalls != 1 {
		t.Errorf("expected 1 start call, got %d", notifier.startCalls)
	}
}

func TestRenewalPreservesOriginalStartedAt(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	now := time.Now()
	originalStartedAt := now.Add(-8 * time.Hour).UnixMilli()

	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Long Task"
		s.StartedAt = originalStartedAt
		s.LiveActivityStartedAt = now.Add(-7*time.Hour - 46*time.Minute)
		s.UpdateToken = "update-token"
		s.PushToStartToken = "pts-token"
	})

	var capturedStartedAt int64
	notifier := &recordingNotifier{
		onStart: func(token, title string, startedAt int64) {
			capturedStartedAt = startedAt
		},
	}
	rn := NewRenewal(store, notifier)
	rn.now = func() time.Time { return now }
	rn.check()

	if capturedStartedAt != originalStartedAt {
		t.Errorf("expected original startedAt %d, got %d", originalStartedAt, capturedStartedAt)
	}

	// LiveActivityStartedAt should be reset
	state := store.Get()
	if state.LiveActivityStartedAt.Before(now.Add(-1 * time.Second)) {
		t.Error("expected LiveActivityStartedAt to be reset to now")
	}
}

type recordingNotifier struct {
	onStart func(token, title string, startedAt int64)
}

func (rn *recordingNotifier) StartActivity(token, title string, startedAt int64) error {
	if rn.onStart != nil {
		rn.onStart(token, title, startedAt)
	}
	return nil
}
func (rn *recordingNotifier) UpdateActivity(token, title string, startedAt int64) error { return nil }
func (rn *recordingNotifier) EndActivity(token string) error                            { return nil }
func (rn *recordingNotifier) SendSilentPush(deviceToken string, taskTitle string, startedAtMs int64) error {
	return nil
}
func (rn *recordingNotifier) SendAlertPush(deviceToken string, title string, body string) error {
	return nil
}
