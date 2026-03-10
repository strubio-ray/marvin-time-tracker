package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// calcDuration computes total tracked milliseconds from a times array.
// The times array is pairs of [start, stop, start, stop, ...].
func calcDuration(times []int64) int64 {
	var total int64
	for i := 0; i+1 < len(times); i += 2 {
		total += times[i+1] - times[i]
	}
	return total
}

type TrackHandler struct {
	store    *StateStore
	marvin   MarvinAPIClient
	notifier Notifier
	broker   *Broker
	history  *HistoryStore
}

func NewTrackHandler(store *StateStore, marvin MarvinAPIClient, notifier Notifier, broker *Broker, history *HistoryStore) *TrackHandler {
	return &TrackHandler{
		store:    store,
		marvin:   marvin,
		notifier: notifier,
		broker:   broker,
		history:  history,
	}
}

type startRequest struct {
	TaskID string `json:"taskId"`
	Title  string `json:"title"`
}

type stopRequest struct {
	TaskID string `json:"taskId,omitempty"`
}

func (th *TrackHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.TaskID == "" {
		http.Error(w, `{"error":"taskId required"}`, http.StatusBadRequest)
		return
	}

	if err := th.marvin.Track(req.TaskID, "START"); err != nil {
		log.Printf("track/start: marvin error: %v", err)
		http.Error(w, `{"error":"failed to start tracking"}`, http.StatusBadGateway)
		return
	}

	now := time.Now()
	startedAt := now.UnixMilli()

	th.store.Update(func(s *State) {
		s.TrackingTaskID = req.TaskID
		s.TaskTitle = req.Title
		s.StartedAt = startedAt
		s.Times = []int64{startedAt}
		s.LiveActivityStartedAt = now
	})

	log.Printf("track/start: started %s (%s)", req.TaskID, req.Title)

	notifyTrackingStarted(r.Context(), th.store, th.notifier, th.broker, req.Title, startedAt, DefaultSilentPushGracePeriod)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "startedAt": startedAt})
}

func (th *TrackHandler) HandleStop(w http.ResponseWriter, r *http.Request) {
	var req stopRequest
	// Body is optional for stop
	json.NewDecoder(r.Body).Decode(&req)

	state := th.store.Get()
	taskID := req.TaskID
	if taskID == "" {
		taskID = state.TrackingTaskID
	}

	if taskID == "" {
		http.Error(w, `{"error":"no task to stop"}`, http.StatusBadRequest)
		return
	}

	// Update all three Marvin data layers so the web client sees tracking as stopped.
	// Order matters: Retrack must come BEFORE Track(STOP) because /api/retrack
	// may re-set the master tracking state. Track(STOP) must be the final call
	// that touches the master state so it remains cleared.
	stopTime := time.Now().UnixMilli()
	times := append(state.Times, stopTime)

	// Layer 1: Update server-side time tracking data.
	if err := th.marvin.Retrack(taskID, times); err != nil {
		log.Printf("track/stop: retrack error (continuing): %v", err)
	}

	// Layer 2: Clear master tracking state. MUST be after Retrack.
	if err := th.marvin.Track(taskID, "STOP"); err != nil {
		log.Printf("track/stop: marvin error: %v", err)
		http.Error(w, `{"error":"failed to stop tracking"}`, http.StatusBadGateway)
		return
	}

	// Layer 3: Update the CouchDB task document (what the web client reads).
	// Calculate duration as sum of all (stop - start) pairs.
	duration := calcDuration(times)
	now := time.Now().UnixMilli()
	if err := th.marvin.UpdateDoc(taskID, []DocSetter{
		{Key: "times", Val: times},
		{Key: "duration", Val: duration},
		{Key: "fieldUpdates.times", Val: now},
		{Key: "fieldUpdates.duration", Val: now},
	}); err != nil {
		log.Printf("track/stop: doc/update error: %v", err)
	}

	updateToken := state.UpdateToken
	stoppedTaskID := taskID
	th.store.Update(func(s *State) {
		s.TrackingTaskID = ""
		s.TaskTitle = ""
		s.StartedAt = 0
		s.Times = nil
		s.LastStopAt = time.Now()
		s.LiveActivityStartedAt = time.Time{}
		s.UpdateToken = ""
	})

	log.Printf("track/stop: stopped %s", taskID)

	notifyTrackingStopped(th.store, th.notifier, th.broker, updateToken, stoppedTaskID)

	if th.history != nil && state.StartedAt > 0 {
		th.history.Add(SessionRecord{
			TaskID:    taskID,
			Title:     state.TaskTitle,
			StartedAt: state.StartedAt,
			StoppedAt: stopTime,
			Duration:  stopTime - state.StartedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
