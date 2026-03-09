package main

import (
	"context"
	"testing"
	"time"
)

func TestNotifyUsesUpdateToken(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.UpdateToken = "update-token"
		s.PushToStartToken = "pts-token"
		s.DeviceToken = "device-token"
	})

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), store, n, "Task", 1000, 10*time.Millisecond)

	if n.updateCalls != 1 {
		t.Errorf("expected 1 update call, got %d", n.updateCalls)
	}
	if n.startCalls != 0 {
		t.Errorf("expected 0 start calls, got %d", n.startCalls)
	}
	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}
	// Alert fallback should NOT fire when Live Activity was sent
	time.Sleep(50 * time.Millisecond)
	if n.alertPushCalls != 0 {
		t.Errorf("expected 0 alert push calls, got %d", n.alertPushCalls)
	}
}

func TestNotifyUsesPushToStartToken(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.PushToStartToken = "pts-token"
		s.DeviceToken = "device-token"
	})

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), store, n, "Task", 1000, 10*time.Millisecond)

	if n.startCalls != 1 {
		t.Errorf("expected 1 start call, got %d", n.startCalls)
	}
	if n.updateCalls != 0 {
		t.Errorf("expected 0 update calls, got %d", n.updateCalls)
	}
	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}

	// Push-to-start token should be cleared
	state := store.Get()
	if state.PushToStartToken != "" {
		t.Errorf("expected pushToStartToken to be cleared, got %s", state.PushToStartToken)
	}
}

func TestNotifySilentPushThenAlert(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.DeviceToken = "device-token"
		s.TrackingTaskID = "task-1"
	})

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), store, n, "My Task", 1000, 10*time.Millisecond)

	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}

	// Wait for grace period to fire alert
	time.Sleep(50 * time.Millisecond)

	if n.alertPushCalls != 1 {
		t.Errorf("expected 1 alert push call, got %d", n.alertPushCalls)
	}
	if n.lastAlertTitle != "Tracking Started" {
		t.Errorf("expected alert title 'Tracking Started', got %s", n.lastAlertTitle)
	}
	if n.lastAlertBody != "My Task" {
		t.Errorf("expected alert body 'My Task', got %s", n.lastAlertBody)
	}
}

func TestNotifySilentPushSucceeds(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.DeviceToken = "device-token"
		s.TrackingTaskID = "task-1"
	})

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), store, n, "Task", 1000, 50*time.Millisecond)

	if n.silentPushCalls != 1 {
		t.Fatalf("expected 1 silent push call, got %d", n.silentPushCalls)
	}

	// Simulate app waking up and registering an update token
	store.Update(func(s *State) {
		s.UpdateToken = "new-update-token"
	})

	// Wait for grace period
	time.Sleep(100 * time.Millisecond)

	// Alert should NOT have fired
	if n.alertPushCalls != 0 {
		t.Errorf("expected 0 alert push calls (silent push succeeded), got %d", n.alertPushCalls)
	}
}

func TestNotifyTrackingStopsDuringGrace(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.DeviceToken = "device-token"
		s.TrackingTaskID = "task-1"
	})

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), store, n, "Task", 1000, 50*time.Millisecond)

	// Simulate tracking stopped
	store.Update(func(s *State) {
		s.TrackingTaskID = ""
	})

	time.Sleep(100 * time.Millisecond)

	if n.alertPushCalls != 0 {
		t.Errorf("expected 0 alert push calls (tracking stopped), got %d", n.alertPushCalls)
	}
}

func TestNotifyNoTokens(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	n := &mockNotifier{}

	// Should not panic
	notifyTrackingStarted(context.Background(), store, n, "Task", 1000, 10*time.Millisecond)

	if n.updateCalls != 0 || n.startCalls != 0 || n.silentPushCalls != 0 || n.alertPushCalls != 0 {
		t.Error("expected no calls with no tokens")
	}
}

func TestNotifyNilNotifier(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.UpdateToken = "token"
	})

	// Should not panic
	notifyTrackingStarted(context.Background(), store, nil, "Task", 1000, 10*time.Millisecond)
}

func TestNotifyContextCancellation(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.DeviceToken = "device-token"
		s.TrackingTaskID = "task-1"
	})

	ctx, cancel := context.WithCancel(context.Background())
	n := &mockNotifier{}
	notifyTrackingStarted(ctx, store, n, "Task", 1000, 1*time.Second)

	// Cancel before grace period
	cancel()
	time.Sleep(50 * time.Millisecond)

	if n.alertPushCalls != 0 {
		t.Errorf("expected 0 alert push calls after cancellation, got %d", n.alertPushCalls)
	}
}

func TestNotifyTrackingStopped(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.DeviceToken = "device-token"
	})

	n := &mockNotifier{}
	notifyTrackingStopped(store, n, "update-token")

	if n.endCalls != 1 {
		t.Errorf("expected 1 end call, got %d", n.endCalls)
	}
	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}
}

func TestNotifyTrackingStoppedNoUpdateToken(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.DeviceToken = "device-token"
	})

	n := &mockNotifier{}
	notifyTrackingStopped(store, n, "")

	if n.endCalls != 0 {
		t.Errorf("expected 0 end calls, got %d", n.endCalls)
	}
	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}
}

func TestNotifyTrackingStoppedNilNotifier(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	// Should not panic
	notifyTrackingStopped(store, nil, "update-token")
}
