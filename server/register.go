package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type registerPayload struct {
	PushToStartToken string `json:"pushToStartToken,omitempty"`
	UpdateToken      string `json:"updateToken,omitempty"`
}

type RegisterHandler struct {
	store *StateStore
}

func NewRegisterHandler(store *StateStore) *RegisterHandler {
	return &RegisterHandler{store: store}
}

func (rh *RegisterHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var payload registerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if payload.PushToStartToken == "" && payload.UpdateToken == "" {
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
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
