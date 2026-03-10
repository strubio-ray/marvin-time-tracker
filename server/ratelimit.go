package main

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	rate     rate.Limit
	burst    int
	stop     chan struct{}
}

func newRateLimiterStore(r rate.Limit, burst int) *rateLimiterStore {
	store := &rateLimiterStore{
		limiters: make(map[string]*ipLimiter),
		rate:     r,
		burst:    burst,
		stop:     make(chan struct{}),
	}
	go store.cleanup()
	return store
}

func (s *rateLimiterStore) Stop() {
	select {
	case <-s.stop:
		return // already stopped
	default:
		close(s.stop)
	}
}

func (s *rateLimiterStore) getLimiter(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.limiters[ip]; ok {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	limiter := rate.NewLimiter(s.rate, s.burst)
	s.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func (s *rateLimiterStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			for ip, entry := range s.limiters {
				if time.Since(entry.lastSeen) > 5*time.Minute {
					delete(s.limiters, ip)
				}
			}
			s.mu.Unlock()
		}
	}
}

var webhookLimiter = newRateLimiterStore(10, 20)

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !webhookLimiter.getLimiter(ip).Allow() {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
