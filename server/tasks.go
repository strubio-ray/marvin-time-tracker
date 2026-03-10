package main

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type taskCache struct {
	mu     sync.Mutex
	data   []byte
	expiry time.Time
}

func (tc *taskCache) get() ([]byte, bool) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if tc.data != nil && time.Now().Before(tc.expiry) {
		result := make([]byte, len(tc.data))
		copy(result, tc.data)
		return result, true
	}
	return nil, false
}

func (tc *taskCache) set(data []byte) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.data = make([]byte, len(data))
	copy(tc.data, data)
	tc.expiry = time.Now().Add(30 * time.Second)
}

func tasksHandler(marvin MarvinAPIClient) http.HandlerFunc {
	cache := &taskCache{}

	return func(w http.ResponseWriter, r *http.Request) {
		if cached, ok := cache.get(); ok {
			w.Header().Set("Content-Type", "application/json")
			w.Write(cached)
			return
		}

		data, err := marvin.TodayItems()
		if err != nil {
			log.Printf("tasks: marvin error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"failed to fetch tasks from Marvin"}`))
			return
		}

		cache.set(data)

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
