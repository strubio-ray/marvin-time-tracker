package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestAPNsStartPayload(t *testing.T) {
	data, err := marshalAPNsPayload("start", "Test Task", 1772734813781, true)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	aps, ok := payload["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("missing aps dictionary")
	}

	if aps["event"] != "start" {
		t.Errorf("expected event start, got %v", aps["event"])
	}
	if aps["attributes-type"] != "TimeTrackerAttributes" {
		t.Errorf("expected attributes-type TimeTrackerAttributes, got %v", aps["attributes-type"])
	}

	cs, ok := aps["content-state"].(map[string]interface{})
	if !ok {
		t.Fatal("missing content-state")
	}
	if cs["taskTitle"] != "Test Task" {
		t.Errorf("expected taskTitle Test Task, got %v", cs["taskTitle"])
	}
	if cs["isTracking"] != true {
		t.Errorf("expected isTracking true, got %v", cs["isTracking"])
	}

	alert, ok := aps["alert"].(map[string]interface{})
	if !ok {
		t.Fatal("missing alert in start payload")
	}
	if alert["title"] != "Tracking Started" {
		t.Errorf("expected alert title, got %v", alert["title"])
	}
}

func TestAPNsUpdatePayload(t *testing.T) {
	data, err := marshalAPNsPayload("update", "Updated Task", 1772734813781, true)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	json.Unmarshal(data, &payload)

	aps := payload["aps"].(map[string]interface{})

	if aps["event"] != "update" {
		t.Errorf("expected event update, got %v", aps["event"])
	}
	// Update should NOT have attributes-type or alert
	if _, ok := aps["attributes-type"]; ok {
		t.Error("update payload should not have attributes-type")
	}
	if _, ok := aps["alert"]; ok {
		t.Error("update payload should not have alert")
	}
}

func TestAPNsSilentPushPayload(t *testing.T) {
	data, err := marshalSilentPushPayload("Test Task", 1772734813781)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	aps, ok := payload["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("missing aps dictionary")
	}

	ca, ok := aps["content-available"]
	if !ok {
		t.Fatal("missing content-available in aps")
	}
	if ca != float64(1) {
		t.Errorf("expected content-available 1, got %v", ca)
	}

	if payload["action"] != "refreshStatus" {
		t.Errorf("expected action refreshStatus, got %v", payload["action"])
	}
	if payload["taskTitle"] != "Test Task" {
		t.Errorf("expected taskTitle Test Task, got %v", payload["taskTitle"])
	}
	if payload["startedAt"] != float64(1772734813781) {
		t.Errorf("expected startedAt 1772734813781, got %v", payload["startedAt"])
	}
}

func TestAPNsAlertPushPayload(t *testing.T) {
	data, err := marshalAlertPushPayload("Tracking Started", "My Task")
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	aps, ok := payload["aps"].(map[string]interface{})
	if !ok {
		t.Fatal("missing aps dictionary")
	}

	alert, ok := aps["alert"].(map[string]interface{})
	if !ok {
		t.Fatal("missing alert in aps")
	}
	if alert["title"] != "Tracking Started" {
		t.Errorf("expected title 'Tracking Started', got %v", alert["title"])
	}
	if alert["body"] != "My Task" {
		t.Errorf("expected body 'My Task', got %v", alert["body"])
	}

	if aps["sound"] != "default" {
		t.Errorf("expected sound 'default', got %v", aps["sound"])
	}
}

func TestAPNsEndPayload(t *testing.T) {
	data, err := marshalAPNsPayload("end", "", 1772734813781, false)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var payload map[string]interface{}
	json.Unmarshal(data, &payload)

	aps := payload["aps"].(map[string]interface{})

	if aps["event"] != "end" {
		t.Errorf("expected event end, got %v", aps["event"])
	}
	if _, ok := aps["dismissal-date"]; !ok {
		t.Error("end payload should have dismissal-date")
	}

	cs := aps["content-state"].(map[string]interface{})
	if cs["isTracking"] != false {
		t.Errorf("expected isTracking false, got %v", cs["isTracking"])
	}
}

func TestMsToAppleSeconds(t *testing.T) {
	// 1772734813781 ms = 1772734813.781 Unix seconds
	// Apple epoch offset = 978307200
	// Expected: 1772734813.781 - 978307200 = 794427613.781
	result := msToAppleSeconds(1772734813781)
	expected := 794427613.781
	if diff := result - expected; diff > 0.001 || diff < -0.001 {
		t.Errorf("msToAppleSeconds(1772734813781) = %f, want ~%f", result, expected)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{400, false},
		{403, false},
		{410, false},
		{500, true},
		{503, true},
	}
	for _, tt := range tests {
		if got := isRetryable(tt.code); got != tt.want {
			t.Errorf("isRetryable(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

// generateTestP8Key creates a temporary P8 ECDSA key file for testing.
func generateTestP8Key(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal key: %v", err)
	}

	pemBlock := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	path := filepath.Join(t.TempDir(), "test.p8")
	if err := os.WriteFile(path, pemBlock, 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	return path, key
}

func TestNewAPNsClient(t *testing.T) {
	keyPath, _ := generateTestP8Key(t)

	client, err := NewAPNsClient(keyPath, "KEYID123", "TEAMID123", "com.test.app", "development")
	if err != nil {
		t.Fatalf("NewAPNsClient error: %v", err)
	}

	if client.host != apnsHostDevelopment {
		t.Errorf("expected development host, got %s", client.host)
	}

	clientProd, err := NewAPNsClient(keyPath, "KEYID123", "TEAMID123", "com.test.app", "production")
	if err != nil {
		t.Fatalf("NewAPNsClient production error: %v", err)
	}
	if clientProd.host != apnsHostProduction {
		t.Errorf("expected production host, got %s", clientProd.host)
	}
}

func TestAPNsJWTCaching(t *testing.T) {
	keyPath, _ := generateTestP8Key(t)

	client, err := NewAPNsClient(keyPath, "KEYID123", "TEAMID123", "com.test.app", "development")
	if err != nil {
		t.Fatalf("NewAPNsClient error: %v", err)
	}

	jwt1, err := client.getJWT()
	if err != nil {
		t.Fatalf("getJWT error: %v", err)
	}

	jwt2, err := client.getJWT()
	if err != nil {
		t.Fatalf("getJWT error: %v", err)
	}

	if jwt1 != jwt2 {
		t.Error("expected JWT to be cached, got different tokens")
	}
}

func TestAPNsRetryOn503(t *testing.T) {
	var attempts atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"reason":"ServiceUnavailable"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	keyPath, key := generateTestP8Key(t)
	_ = keyPath

	client := &APNsClient{
		httpClient: ts.Client(),
		keyID:      "KEYID123",
		teamID:     "TEAMID123",
		privateKey: key,
		bundleID:   "com.test.app",
		host:       ts.URL,
	}

	err := client.send("testtoken1234567890", map[string]interface{}{"test": true}, 10, "com.test.app", "alert")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestAPNsNoRetryOn400(t *testing.T) {
	var attempts atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"reason":"BadDeviceToken"}`))
	}))
	defer ts.Close()

	_, key := generateTestP8Key(t)

	client := &APNsClient{
		httpClient: ts.Client(),
		keyID:      "KEYID123",
		teamID:     "TEAMID123",
		privateKey: key,
		bundleID:   "com.test.app",
		host:       ts.URL,
	}

	err := client.send("testtoken1234567890", map[string]interface{}{"test": true}, 10, "com.test.app", "alert")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	if got := attempts.Load(); got != 1 {
		t.Errorf("expected 1 attempt (no retry on 4xx), got %d", got)
	}
}
