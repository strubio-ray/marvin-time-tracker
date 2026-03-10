package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const sseKeepaliveInterval = 20 * time.Second

const sseWriteDeadline = 60 * time.Second

type sseStateEvent struct {
	Tracking  bool   `json:"tracking"`
	TaskID    string `json:"taskId,omitempty"`
	TaskTitle string `json:"taskTitle,omitempty"`
	StartedAt int64  `json:"startedAt,omitempty"`
}

func sseHandler(store *StateStore, broker *Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)

		// Clear the server-wide write deadline for this long-lived connection
		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ch, unsub := broker.Subscribe()
		defer unsub()

		// Send initial state snapshot
		state := store.Get()
		initial := sseStateEvent{
			Tracking: state.IsTracking(),
		}
		if state.IsTracking() {
			initial.TaskID = state.TrackingTaskID
			initial.TaskTitle = state.TaskTitle
			initial.StartedAt = state.StartedAt
		}

		data, err := json.Marshal(initial)
		if err != nil {
			log.Printf("sse: marshal error: %v", err)
			return
		}

		rc.SetWriteDeadline(time.Now().Add(sseWriteDeadline))
		if _, err := fmt.Fprintf(w, "event: state\ndata: %s\n\n", data); err != nil {
			log.Printf("sse: write error: %v", err)
			return
		}
		if err := rc.Flush(); err != nil {
			log.Printf("sse: flush error: %v", err)
			return
		}

		ticker := time.NewTicker(sseKeepaliveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-ch:
				rc.SetWriteDeadline(time.Now().Add(sseWriteDeadline))
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data); err != nil {
					log.Printf("sse: write error: %v", err)
					return
				}
				if err := rc.Flush(); err != nil {
					log.Printf("sse: flush error: %v", err)
					return
				}
			case <-ticker.C:
				rc.SetWriteDeadline(time.Now().Add(sseWriteDeadline))
				if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
					log.Printf("sse: write error: %v", err)
					return
				}
				if err := rc.Flush(); err != nil {
					log.Printf("sse: flush error: %v", err)
					return
				}
			}
		}
	}
}
