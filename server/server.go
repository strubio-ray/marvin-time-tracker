package main

import (
	"encoding/json"
	"net/http"

	"github.com/rs/cors"
)

type Server struct {
	handler http.Handler
	store   *StateStore
}

func NewServer(store *StateStore, dedup *DedupCache, notifier Notifier) *Server {
	wh := NewWebhookHandler(store, dedup, notifier)
	rh := NewRegisterHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", statusHandler(store))
	mux.HandleFunc("POST /webhook/start", wh.HandleStart)
	mux.HandleFunc("POST /webhook/stop", wh.HandleStop)
	mux.HandleFunc("POST /register", rh.Handle)

	c := cors.New(cors.Options{
		AllowedOrigins:     []string{"*"},
		AllowedMethods:     []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowedHeaders:     []string{"Content-Type"},
		OptionsSuccessStatus: 200,
	})

	return &Server{
		handler: c.Handler(mux),
		store:   store,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func statusHandler(store *StateStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := store.Get()

		resp := map[string]interface{}{
			"status":              "ok",
			"tracking":           state.IsTracking(),
			"hasPushToStartToken": state.PushToStartToken != "",
			"hasUpdateToken":      state.UpdateToken != "",
		}

		if state.IsTracking() {
			resp["taskId"] = state.TrackingTaskID
			resp["taskTitle"] = state.TaskTitle
			resp["startedAt"] = state.StartedAt
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
