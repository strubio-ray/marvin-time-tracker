package main

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSEInitialState(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.TrackingTaskID = "task-1"
		s.TaskTitle = "Test Task"
		s.StartedAt = 1000
	})
	broker := NewBroker()

	srv := httptest.NewServer(sseHandler(store, broker))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}

	scanner := bufio.NewScanner(resp.Body)
	var eventType, data string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && eventType != "" {
			break
		}
	}

	if eventType != "state" {
		t.Errorf("expected event type 'state', got %s", eventType)
	}

	var state sseStateEvent
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !state.Tracking {
		t.Error("expected tracking=true")
	}
	if state.TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", state.TaskID)
	}
	if state.TaskTitle != "Test Task" {
		t.Errorf("expected 'Test Task', got %s", state.TaskTitle)
	}
	if state.StartedAt != 1000 {
		t.Errorf("expected startedAt=1000, got %d", state.StartedAt)
	}
}

func TestSSEInitialStateIdle(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	broker := NewBroker()

	srv := httptest.NewServer(sseHandler(store, broker))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var data string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && data != "" {
			break
		}
	}

	var state sseStateEvent
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if state.Tracking {
		t.Error("expected tracking=false for idle state")
	}
}

func TestSSEReceivesBroadcast(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	broker := NewBroker()

	srv := httptest.NewServer(sseHandler(store, broker))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	// Read past initial state event
	for scanner.Scan() {
		if scanner.Text() == "" {
			break
		}
	}

	// Broadcast a tracking_started event
	broker.Broadcast(SSEEvent{
		Type: "tracking_started",
		Data: []byte(`{"taskId":"task-2","taskTitle":"New Task","startedAt":2000}`),
	})

	var eventType, data string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && eventType != "" {
			break
		}
	}

	if eventType != "tracking_started" {
		t.Errorf("expected event type 'tracking_started', got %s", eventType)
	}
	if !strings.Contains(data, "task-2") {
		t.Errorf("expected data to contain 'task-2', got %s", data)
	}
}

func TestSSEDisconnectCleansUp(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	broker := NewBroker()

	srv := httptest.NewServer(sseHandler(store, broker))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}

	// Read initial event to ensure subscription is active
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if scanner.Text() == "" {
			break
		}
	}

	// Cancel context to disconnect
	cancel()
	resp.Body.Close()

	// Give handler time to clean up
	time.Sleep(100 * time.Millisecond)

	if broker.ClientCount() != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", broker.ClientCount())
	}
}

func TestSSEWriteErrorCleansUp(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	broker := NewBroker()

	srv := httptest.NewServer(sseHandler(store, broker))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}

	// Read initial event to ensure subscription is active
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if scanner.Text() == "" {
			break
		}
	}

	if broker.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", broker.ClientCount())
	}

	// Close the connection from the client side to cause a write error
	resp.Body.Close()

	// Broadcast an event — the handler should detect the write error and clean up
	broker.Broadcast(SSEEvent{
		Type: "tracking_started",
		Data: []byte(`{"taskId":"task-x","taskTitle":"Write Error","startedAt":9999}`),
	})

	// Give handler time to detect the write error and clean up
	time.Sleep(200 * time.Millisecond)

	if broker.ClientCount() != 0 {
		t.Errorf("expected 0 clients after write error, got %d", broker.ClientCount())
	}
}
