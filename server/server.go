package main

import (
	"encoding/json"
	"net/http"
	"strconv"

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

	wh := NewWebhookHandler(store, dedup, notifier, so.broker, so.history)
	rh := NewRegisterHandler(store, notifier, so.broker)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", statusHandler(store))
	mux.HandleFunc("POST /webhook/start", wh.HandleStart)
	mux.HandleFunc("POST /webhook/stop", wh.HandleStop)
	mux.HandleFunc("POST /register", rh.Handle)

	if so.history != nil {
		mux.HandleFunc("GET /history", historyHandler(so.history))
	}

	if so.broker != nil {
		mux.HandleFunc("GET /events", sseHandler(store, so.broker))
	}

	if so.marvin != nil {
		th := NewTrackHandler(store, so.marvin, notifier, so.broker, so.history)
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
	marvin  MarvinAPIClient
	broker  *Broker
	history *HistoryStore
}

type ServerOption func(*serverOptions)

func WithBroker(b *Broker) ServerOption {
	return func(so *serverOptions) {
		so.broker = b
	}
}

func WithMarvinClient(mc MarvinAPIClient) ServerOption {
	return func(so *serverOptions) {
		so.marvin = mc
	}
}

func WithHistory(h *HistoryStore) ServerOption {
	return func(so *serverOptions) {
		so.history = h
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func historyHandler(history *HistoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 10
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		if limit > 200 {
			limit = 200
		}

		sessions := history.Recent(limit)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	}
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
