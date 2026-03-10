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
