package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type registerPayload struct {
	PushToStartToken string `json:"pushToStartToken,omitempty"`
	UpdateToken      string `json:"updateToken,omitempty"`
	DeviceToken      string `json:"deviceToken,omitempty"`
}

type RegisterHandler struct {
	store    *StateStore
	notifier Notifier
	broker   BrokerPublisher
}

func NewRegisterHandler(store *StateStore, notifier Notifier, broker BrokerPublisher) *RegisterHandler {
	return &RegisterHandler{store: store, notifier: notifier, broker: broker}
}

func (rh *RegisterHandler) Handle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var payload registerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if payload.PushToStartToken == "" && payload.UpdateToken == "" && payload.DeviceToken == "" {
		http.Error(w, `{"error":"no tokens provided"}`, http.StatusBadRequest)
		return
	}

	rh.store.Update(func(s *State) {
		if payload.PushToStartToken != "" {
			s.PushToStartToken = payload.PushToStartToken
			log.Printf("register: updated pushToStartToken")
		}
		if payload.UpdateToken != "" {
			s.UpdateToken = payload.UpdateToken
			log.Printf("register: updated updateToken")
		}
		if payload.DeviceToken != "" {
			s.DeviceToken = payload.DeviceToken
			log.Printf("register: updated deviceToken")
		}
	})

	// If tracking is active, proactively push to the newly registered token.
	// This handles silent push-to-start failures, token rotations, and race conditions.
	state := rh.store.Get()
	if state.IsTracking() {
		if payload.UpdateToken != "" && rh.notifier != nil {
			if err := rh.notifier.UpdateActivity(payload.UpdateToken, state.TaskTitle, state.StartedAt); err != nil {
				log.Printf("register: proactive update error: %v", err)
			} else {
				log.Printf("register: sent proactive update for active tracking")
			}
		} else if payload.DeviceToken != "" && state.UpdateToken == "" && state.PushToStartToken == "" {
			// Device token registered while tracking but no activity tokens — use fallback chain
			tokens := NotifyTokens{
				DeviceToken: payload.DeviceToken,
			}
			notifyTrackingStarted(r.Context(), tokens, rh.notifier, rh.broker, state.TaskTitle, state.StartedAt, DefaultSilentPushGracePeriod, func() string {
				s := rh.store.Get()
				if !s.IsTracking() {
					return "stopped"
				}
				return s.UpdateToken
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
