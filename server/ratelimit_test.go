package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestRateLimitStore(t *testing.T) {
	store := newRateLimiterStore(1, 2) // 1 req/s, burst 2

	// First 2 requests should succeed (burst)
	for i := range 2 {
		if !store.getLimiter("192.168.1.1").Allow() {
			t.Errorf("request %d: expected allowed", i)
		}
	}

	// Third request should be rate limited
	if store.getLimiter("192.168.1.1").Allow() {
		t.Error("expected rate limited after burst")
	}

	// Different IP should still be allowed
	if !store.getLimiter("10.0.0.1").Allow() {
		t.Error("different IP: expected allowed")
	}
}

func TestRateLimitMiddlewareIntegration(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	limited := rateLimitMiddleware(okHandler)

	// Webhook limiter has burst of 20, so first 20 should pass
	for i := range 20 {
		req := httptest.NewRequest(http.MethodPost, "/webhook/start", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		limited(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	// Next request from same IP should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/webhook/start", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	limited(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after burst, got %d", w.Code)
	}
}

func TestRateLimitIPExtraction(t *testing.T) {
	// Verify net.SplitHostPort works as expected
	ip, _, err := net.SplitHostPort("192.168.1.1:12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ip != "192.168.1.1" {
		t.Errorf("expected 192.168.1.1, got %s", ip)
	}
}

func TestGetLimiter(t *testing.T) {
	store := newRateLimiterStore(rate.Limit(10), 20)

	l1 := store.getLimiter("1.2.3.4")
	l2 := store.getLimiter("1.2.3.4")

	if l1 != l2 {
		t.Error("expected same limiter for same IP")
	}

	l3 := store.getLimiter("5.6.7.8")
	if l1 == l3 {
		t.Error("expected different limiter for different IP")
	}
}
