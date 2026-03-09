package main

import (
	"encoding/json"
	"testing"
)

func TestAPNsStartPayload(t *testing.T) {
	data, err := marshalAPNsPayload("start", "Test Task", 1772734813781, true)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	aps, ok := payload["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("missing aps dictionary")
	}

	if aps["event"] != "start" {
		t.Errorf("expected event start, got %v", aps["event"])
	}
	if aps["attributes-type"] != "TimeTrackerAttributes" {
		t.Errorf("expected attributes-type TimeTrackerAttributes, got %v", aps["attributes-type"])
	}

	cs, ok := aps["content-state"].(map[string]interface{})
	if !ok {
		t.Fatal("missing content-state")
	}
	if cs["taskTitle"] != "Test Task" {
		t.Errorf("expected taskTitle Test Task, got %v", cs["taskTitle"])
	}
	if cs["isTracking"] != true {
		t.Errorf("expected isTracking true, got %v", cs["isTracking"])
	}

	alert, ok := aps["alert"].(map[string]interface{})
	if !ok {
		t.Fatal("missing alert in start payload")
	}
	if alert["title"] != "Tracking Started" {
		t.Errorf("expected alert title, got %v", alert["title"])
	}
}

func TestAPNsUpdatePayload(t *testing.T) {
	data, err := marshalAPNsPayload("update", "Updated Task", 1772734813781, true)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	json.Unmarshal(data, &payload)

	aps := payload["aps"].(map[string]interface{})

	if aps["event"] != "update" {
		t.Errorf("expected event update, got %v", aps["event"])
	}
	// Update should NOT have attributes-type or alert
	if _, ok := aps["attributes-type"]; ok {
		t.Error("update payload should not have attributes-type")
	}
	if _, ok := aps["alert"]; ok {
		t.Error("update payload should not have alert")
	}
}

func TestAPNsSilentPushPayload(t *testing.T) {
	data, err := marshalSilentPushPayload("Test Task", 1772734813781)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	aps, ok := payload["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("missing aps dictionary")
	}

	ca, ok := aps["content-available"]
	if !ok {
		t.Fatal("missing content-available in aps")
	}
	if ca != float64(1) {
		t.Errorf("expected content-available 1, got %v", ca)
	}

	if payload["action"] != "refreshStatus" {
		t.Errorf("expected action refreshStatus, got %v", payload["action"])
	}
	if payload["taskTitle"] != "Test Task" {
		t.Errorf("expected taskTitle Test Task, got %v", payload["taskTitle"])
	}
	if payload["startedAt"] != float64(1772734813781) {
		t.Errorf("expected startedAt 1772734813781, got %v", payload["startedAt"])
	}
}

func TestAPNsAlertPushPayload(t *testing.T) {
	data, err := marshalAlertPushPayload("Tracking Started", "My Task")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	aps, ok := payload["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("missing aps dictionary")
	}

	alert, ok := aps["alert"].(map[string]interface{})
	if !ok {
		t.Fatal("missing alert in aps")
	}
	if alert["title"] != "Tracking Started" {
		t.Errorf("expected title 'Tracking Started', got %v", alert["title"])
	}
	if alert["body"] != "My Task" {
		t.Errorf("expected body 'My Task', got %v", alert["body"])
	}

	if aps["sound"] != "default" {
		t.Errorf("expected sound 'default', got %v", aps["sound"])
	}
}

func TestAPNsEndPayload(t *testing.T) {
	data, err := marshalAPNsPayload("end", "", 1772734813781, false)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	json.Unmarshal(data, &payload)

	aps := payload["aps"].(map[string]interface{})

	if aps["event"] != "end" {
		t.Errorf("expected event end, got %v", aps["event"])
	}
	if _, ok := aps["dismissal-date"]; !ok {
		t.Error("end payload should have dismissal-date")
	}

	cs := aps["content-state"].(map[string]interface{})
	if cs["isTracking"] != false {
		t.Errorf("expected isTracking false, got %v", cs["isTracking"])
	}
}
