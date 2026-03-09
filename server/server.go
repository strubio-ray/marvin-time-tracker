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

func NewServer(store *StateStore, dedup *DedupCache, notifier Notifier, opts ...ServerOption) *Server {
	so := &serverOptions{}
	for _, o := range opts {
		o(so)
	}

	wh := NewWebhookHandler(store, dedup, notifier)
	rh := NewRegisterHandler(store, notifier)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", statusHandler(store))
	mux.HandleFunc("POST /webhook/start", wh.HandleStart)
	mux.HandleFunc("POST /webhook/stop", wh.HandleStop)
	mux.HandleFunc("POST /register", rh.Handle)

	if so.marvin != nil {
		th := NewTrackHandler(store, so.marvin, notifier)
		mux.HandleFunc("POST /start", th.HandleStart)
		mux.HandleFunc("POST /stop", th.HandleStop)
	}

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

type serverOptions struct {
	marvin MarvinAPIClient
}

type ServerOption func(*serverOptions)

func WithMarvinClient(mc MarvinAPIClient) ServerOption {
	return func(so *serverOptions) {
		so.marvin = mc
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
			"hasDeviceToken":      state.DeviceToken != "",
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
