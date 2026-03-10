package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// requireAPIKey wraps a handler with Bearer token authentication.
// If apiKey is empty, the handler is passed through without auth (dev mode).
func requireAPIKey(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			next(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeAuthError(w, "missing Authorization header")
			return
		}

		token, ok := strings.CutPrefix(auth, "Bearer ")
		if !ok {
			writeAuthError(w, "invalid Authorization format, expected Bearer token")
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
			writeAuthError(w, "invalid API key")
			return
		}

		next(w, r)
	}
}

func writeAuthError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
