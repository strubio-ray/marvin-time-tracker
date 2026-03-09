package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type webhookPayload struct {
	TaskID    string  `json:"_id"`
	Title     string  `json:"title"`
	Timestamp int64   `json:"timestamp"`
	Times     []int64 `json:"times"`
}

type WebhookHandler struct {
	store    *StateStore
	dedup    *DedupCache
	notifier Notifier
}

func NewWebhookHandler(store *StateStore, dedup *DedupCache, notifier Notifier) *WebhookHandler {
	return &WebhookHandler{
		store:    store,
		dedup:    dedup,
		notifier: notifier,
	}
}

func (wh *WebhookHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	// Acknowledge immediately
	w.WriteHeader(http.StatusOK)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("webhook/start: failed to read body: %v", err)
		return
	}
	log.Printf("webhook/start: raw body: %s", string(body))

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("webhook/start: invalid JSON: %v", err)
		return
	}

	if payload.TaskID == "" {
		log.Printf("webhook/start: missing taskId")
		return
	}

	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().UnixMilli()
	}

	key := DedupKey("start", payload.TaskID, payload.Timestamp)
	if wh.dedup.IsDuplicate(key) {
		log.Printf("webhook/start: dedup hit for %s", payload.TaskID)
		return
	}

	// Bounce-back detection: if this webhook re-starts a task we just stopped
	// from the iOS app, it's the Marvin web client resuming from browser-local
	// state on page refresh. We can't prevent this client-side behavior, but we
	// must not re-enter tracking state on our server.
	state := wh.store.Get()
	if payload.TaskID == state.LastStoppedTaskID {
		log.Printf("webhook/start: ignoring bounce-back for %s (Marvin web client resumed from local state)", payload.TaskID)
		return
	}

	// If the webhook includes a times array, check if tracking is actually active.
	// Marvin uses paired timestamps [start, stop, start, stop, ...].
	// Odd count = tracking active, even count = tracking stopped (stale webhook).
	if len(payload.Times) > 0 && len(payload.Times)%2 == 0 {
		log.Printf("webhook/start: ignoring stale webhook for %s (times has %d entries, tracking not active)", payload.TaskID, len(payload.Times))
		return
	}

	now := time.Now()
	wh.store.Update(func(s *State) {
		s.TrackingTaskID = payload.TaskID
		s.TaskTitle = payload.Title
		s.StartedAt = payload.Timestamp
		s.Times = payload.Times
		s.LastWebhookAt = now
		s.LiveActivityStartedAt = now
	})

	log.Printf("webhook/start: tracking %s (%s)", payload.TaskID, payload.Title)

	notifyTrackingStarted(r.Context(), wh.store, wh.notifier, payload.Title, payload.Timestamp, DefaultSilentPushGracePeriod)
}

func (wh *WebhookHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("webhook/stop: failed to read body: %v", err)
		return
	}
	log.Printf("webhook/stop: raw body: %s", string(body))

	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("webhook/stop: invalid JSON: %v", err)
		return
	}

	if payload.Timestamp == 0 {
		payload.Timestamp = time.Now().UnixMilli()
	}

	if payload.TaskID != "" {
		key := DedupKey("stop", payload.TaskID, payload.Timestamp)
		if wh.dedup.IsDuplicate(key) {
			log.Printf("webhook/stop: dedup hit for %s", payload.TaskID)
			return
		}
	}

	state := wh.store.Get()
	updateToken := state.UpdateToken

	now := time.Now()
	wh.store.Update(func(s *State) {
		s.TrackingTaskID = ""
		s.TaskTitle = ""
		s.StartedAt = 0
		s.Times = nil
		s.LastWebhookAt = now
		s.LastStopAt = now
		s.LastStoppedTaskID = ""
		s.LiveActivityStartedAt = time.Time{}
		s.UpdateToken = ""
	})

	log.Printf("webhook/stop: stopped tracking")

	notifyTrackingStopped(wh.store, wh.notifier, updateToken)
}
