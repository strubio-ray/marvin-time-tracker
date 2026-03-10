package main

import (
	"log"
	"net/http"
)

func tasksHandler(marvin MarvinAPIClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := marvin.TodayItems()
		if err != nil {
			log.Printf("tasks: marvin error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error":"failed to fetch tasks from Marvin"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
