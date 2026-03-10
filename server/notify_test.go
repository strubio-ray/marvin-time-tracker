package main

import (
	"context"
	"testing"
	"time"
)

func TestNotifyUsesUpdateToken(t *testing.T) {
	tokens := NotifyTokens{
		UpdateToken:      "update-token",
		PushToStartToken: "pts-token",
		DeviceToken:      "device-token",
	}

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), tokens, n, nil, "Task", 1000, 10*time.Millisecond, func() string { return "" })

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
	tokens := NotifyTokens{
		PushToStartToken: "pts-token",
		DeviceToken:      "device-token",
	}

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), tokens, n, nil, "Task", 1000, 10*time.Millisecond, func() string { return "" })

	if n.startCalls != 1 {
		t.Errorf("expected 1 start call, got %d", n.startCalls)
	}
	if n.updateCalls != 0 {
		t.Errorf("expected 0 update calls, got %d", n.updateCalls)
	}
	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}
}

func TestNotifySilentPushThenAlert(t *testing.T) {
	tokens := NotifyTokens{
		DeviceToken: "device-token",
	}

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), tokens, n, nil, "My Task", 1000, 10*time.Millisecond, func() string { return "" })

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
	tokens := NotifyTokens{
		DeviceToken: "device-token",
	}

	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), tokens, n, nil, "Task", 1000, 50*time.Millisecond, func() string {
		return "new-update-token"
	})

	if n.silentPushCalls != 1 {
		t.Fatalf("expected 1 silent push call, got %d", n.silentPushCalls)
	}

	// Wait for grace period
	time.Sleep(100 * time.Millisecond)

	// Alert should NOT have fired
	if n.alertPushCalls != 0 {
		t.Errorf("expected 0 alert push calls (silent push succeeded), got %d", n.alertPushCalls)
	}
}

func TestNotifyTrackingStopsDuringGrace(t *testing.T) {
	tokens := NotifyTokens{
		DeviceToken: "device-token",
	}

	// Return "stopped" to simulate tracking having stopped
	n := &mockNotifier{}
	notifyTrackingStarted(context.Background(), tokens, n, nil, "Task", 1000, 50*time.Millisecond, func() string { return "stopped" })

	time.Sleep(100 * time.Millisecond)

	if n.alertPushCalls != 0 {
		t.Errorf("expected 0 alert push calls (tracking stopped), got %d", n.alertPushCalls)
	}
}

func TestNotifyNoTokens(t *testing.T) {
	tokens := NotifyTokens{}
	n := &mockNotifier{}

	// Should not panic
	notifyTrackingStarted(context.Background(), tokens, n, nil, "Task", 1000, 10*time.Millisecond, nil)

	if n.updateCalls != 0 || n.startCalls != 0 || n.silentPushCalls != 0 || n.alertPushCalls != 0 {
		t.Error("expected no calls with no tokens")
	}
}

func TestNotifyNilNotifier(t *testing.T) {
	tokens := NotifyTokens{
		UpdateToken: "token",
	}

	// Should not panic
	notifyTrackingStarted(context.Background(), tokens, nil, nil, "Task", 1000, 10*time.Millisecond, nil)
}

func TestNotifyContextCancellation(t *testing.T) {
	tokens := NotifyTokens{
		DeviceToken: "device-token",
	}

	ctx, cancel := context.WithCancel(context.Background())
	n := &mockNotifier{}
	notifyTrackingStarted(ctx, tokens, n, nil, "Task", 1000, 1*time.Second, func() string { return "" })

	// Cancel before grace period
	cancel()
	time.Sleep(50 * time.Millisecond)

	if n.alertPushCalls != 0 {
		t.Errorf("expected 0 alert push calls after cancellation, got %d", n.alertPushCalls)
	}
}

func TestNotifyTrackingStopped(t *testing.T) {
	n := &mockNotifier{}
	notifyTrackingStopped(n, nil, "update-token", "device-token", "")

	if n.endCalls != 1 {
		t.Errorf("expected 1 end call, got %d", n.endCalls)
	}
	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}
}

func TestNotifyTrackingStoppedNoUpdateToken(t *testing.T) {
	n := &mockNotifier{}
	notifyTrackingStopped(n, nil, "", "device-token", "")

	if n.endCalls != 0 {
		t.Errorf("expected 0 end calls, got %d", n.endCalls)
	}
	if n.silentPushCalls != 1 {
		t.Errorf("expected 1 silent push call, got %d", n.silentPushCalls)
	}
}

func TestNotifyTrackingStoppedNilNotifier(t *testing.T) {
	// Should not panic
	notifyTrackingStopped(nil, nil, "update-token", "device-token", "")
}
