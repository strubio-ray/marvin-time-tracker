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
	broker   *Broker
	history  *HistoryStore
	debug    bool
}

func NewWebhookHandler(store *StateStore, dedup *DedupCache, notifier Notifier, broker *Broker, history *HistoryStore, debug bool) *WebhookHandler {
	return &WebhookHandler{
		store:    store,
		dedup:    dedup,
		notifier: notifier,
		broker:   broker,
		history:  history,
		debug:    debug,
	}
}

func (wh *WebhookHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	// Acknowledge immediately
	w.WriteHeader(http.StatusOK)

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("webhook/start: failed to read body: %v", err)
		return
	}
	if wh.debug {
		log.Printf("webhook/start: raw body: %s", string(body))
	}

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

	state := wh.store.Get()
	tokens := NotifyTokens{
		UpdateToken:      state.UpdateToken,
		PushToStartToken: state.PushToStartToken,
		DeviceToken:      state.DeviceToken,
	}
	if tokens.PushToStartToken != "" {
		wh.store.Update(func(s *State) { s.PushToStartToken = "" })
	}
	notifyTrackingStarted(r.Context(), tokens, wh.notifier, wh.broker, payload.Title, payload.Timestamp, DefaultSilentPushGracePeriod, func() string {
		s := wh.store.Get()
		if !s.IsTracking() {
			return "stopped"
		}
		return s.UpdateToken
	})
}

func (wh *WebhookHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("webhook/stop: failed to read body: %v", err)
		return
	}
	if wh.debug {
		log.Printf("webhook/stop: raw body: %s", string(body))
	}

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

	now := time.Now()
	wh.store.Update(func(s *State) {
		s.LastWebhookAt = now
	})
	prev, _ := wh.store.ClearTracking(now)
	updateToken := prev.UpdateToken
	stoppedTaskID := prev.TrackingTaskID

	log.Printf("webhook/stop: stopped tracking")

	notifyTrackingStopped(wh.notifier, wh.broker, updateToken, prev.DeviceToken, stoppedTaskID)

	if wh.history != nil && prev.StartedAt > 0 {
		stopMs := time.Now().UnixMilli()
		wh.history.Add(SessionRecord{
			TaskID:    stoppedTaskID,
			Title:     prev.TaskTitle,
			StartedAt: prev.StartedAt,
			StoppedAt: stopMs,
			Duration:  stopMs - prev.StartedAt,
		})
	}
}
