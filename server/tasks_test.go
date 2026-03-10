package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTasksHandler_Success(t *testing.T) {
	mc := &mockMarvinClient{
		todayItemsResult: []byte(`[{"_id":"t1","title":"Task 1"}]`),
	}

	handler := tasksHandler(mc)
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	if w.Body.String() != `[{"_id":"t1","title":"Task 1"}]` {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestTasksHandler_CacheHit(t *testing.T) {
	mc := &mockMarvinClient{
		todayItemsResult: []byte(`[{"_id":"t1","title":"Task 1"}]`),
	}

	handler := tasksHandler(mc)

	// First request — cache miss
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w.Code)
	}
	if mc.todayItemsCalls != 1 {
		t.Fatalf("expected 1 Marvin call, got %d", mc.todayItemsCalls)
	}

	// Second request — cache hit, should not call Marvin again
	req = httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w = httptest.NewRecorder()
	handler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("second request: expected 200, got %d", w.Code)
	}
	if mc.todayItemsCalls != 1 {
		t.Errorf("expected 1 Marvin call (cached), got %d", mc.todayItemsCalls)
	}
	if w.Body.String() != `[{"_id":"t1","title":"Task 1"}]` {
		t.Errorf("unexpected cached body: %s", w.Body.String())
	}
}

func TestTasksHandler_MarvinError(t *testing.T) {
	mc := &mockMarvinClient{
		todayItemsErr: true,
	}

	handler := tasksHandler(mc)
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}
