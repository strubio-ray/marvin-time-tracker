package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterPushToStartToken(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	rh := NewRegisterHandler(store)

	body, _ := json.Marshal(registerPayload{PushToStartToken: "token-abc"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	rh.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	state := store.Get()
	if state.PushToStartToken != "token-abc" {
		t.Errorf("expected token-abc, got %s", state.PushToStartToken)
	}
}

func TestRegisterUpdateToken(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	rh := NewRegisterHandler(store)

	body, _ := json.Marshal(registerPayload{UpdateToken: "update-xyz"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	rh.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	state := store.Get()
	if state.UpdateToken != "update-xyz" {
		t.Errorf("expected update-xyz, got %s", state.UpdateToken)
	}
}

func TestRegisterBothTokens(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	rh := NewRegisterHandler(store)

	body, _ := json.Marshal(registerPayload{
		PushToStartToken: "start-1",
		UpdateToken:      "update-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	rh.Handle(w, req)

	state := store.Get()
	if state.PushToStartToken != "start-1" || state.UpdateToken != "update-1" {
		t.Errorf("expected both tokens set, got pts=%s ut=%s", state.PushToStartToken, state.UpdateToken)
	}
}

func TestRegisterTokenReplacement(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	store.Update(func(s *State) {
		s.PushToStartToken = "old-token"
	})

	rh := NewRegisterHandler(store)
	body, _ := json.Marshal(registerPayload{PushToStartToken: "new-token"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	rh.Handle(w, req)

	state := store.Get()
	if state.PushToStartToken != "new-token" {
		t.Errorf("expected new-token, got %s", state.PushToStartToken)
	}
}

func TestRegisterNoTokens(t *testing.T) {
	store := NewStateStore(tempStateFile(t))
	rh := NewRegisterHandler(store)

	body, _ := json.Marshal(registerPayload{})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	w := httptest.NewRecorder()
	rh.Handle(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
