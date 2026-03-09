package main

import (
	"sync"
	"testing"
	"time"
)

func TestDedupKey(t *testing.T) {
	// Same 15-second bucket
	k1 := DedupKey("start", "task-1", 15000)
	k2 := DedupKey("start", "task-1", 15001)
	if k1 != k2 {
		t.Errorf("expected same key within 15s bucket, got %s and %s", k1, k2)
	}

	// Different bucket
	k3 := DedupKey("start", "task-1", 30000)
	if k1 == k3 {
		t.Errorf("expected different key for different bucket")
	}

	// Different task
	k4 := DedupKey("start", "task-2", 15000)
	if k1 == k4 {
		t.Errorf("expected different key for different task")
	}

	// Different event type, same task and bucket
	k5 := DedupKey("stop", "task-1", 15000)
	if k1 == k5 {
		t.Errorf("expected different key for different event type")
	}
}

func TestDedupCacheBasic(t *testing.T) {
	dc := NewDedupCache(60 * time.Second)

	if dc.IsDuplicate("key-1") {
		t.Error("first call should not be duplicate")
	}
	if !dc.IsDuplicate("key-1") {
		t.Error("second call should be duplicate")
	}
	if dc.IsDuplicate("key-2") {
		t.Error("different key should not be duplicate")
	}
}

func TestDedupCacheTTLExpiry(t *testing.T) {
	now := time.Now()
	dc := NewDedupCache(60 * time.Second)
	dc.now = func() time.Time { return now }

	dc.IsDuplicate("key-1")

	// Advance past TTL
	dc.now = func() time.Time { return now.Add(61 * time.Second) }

	if dc.IsDuplicate("key-1") {
		t.Error("key should be evicted after TTL")
	}
}

func TestDedupCacheConcurrent(t *testing.T) {
	dc := NewDedupCache(60 * time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dc.IsDuplicate("key-concurrent")
		}()
	}
	wg.Wait()
}
