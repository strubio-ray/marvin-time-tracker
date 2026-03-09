package main

import (
	"strconv"
	"sync"
	"time"
)

type dedupEntry struct {
	expiresAt time.Time
}

type DedupCache struct {
	mu      sync.Mutex
	entries map[string]dedupEntry
	ttl     time.Duration
	now     func() time.Time // for testing
}

func NewDedupCache(ttl time.Duration) *DedupCache {
	return &DedupCache{
		entries: make(map[string]dedupEntry),
		ttl:     ttl,
		now:     time.Now,
	}
}

// DedupKey creates a composite key that collapses events within a 15-second window.
func DedupKey(event string, taskID string, timestampMs int64) string {
	bucket := timestampMs / 15000
	return event + "_" + taskID + "_" + strconv.FormatInt(bucket, 10)
}

// IsDuplicate returns true if this key was already seen within the TTL window.
func (dc *DedupCache) IsDuplicate(key string) bool {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := dc.now()
	dc.evictLocked(now)

	if _, exists := dc.entries[key]; exists {
		return true
	}

	dc.entries[key] = dedupEntry{expiresAt: now.Add(dc.ttl)}
	return false
}

func (dc *DedupCache) evictLocked(now time.Time) {
	for k, e := range dc.entries {
		if now.After(e.expiresAt) {
			delete(dc.entries, k)
		}
	}
}
