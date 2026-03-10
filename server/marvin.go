package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const marvinBaseURL = "https://serv.amazingmarvin.com/api"

// MarvinAPIClient interfaces with the Marvin API.
type MarvinAPIClient interface {
	Track(taskID string, action string) error
	Retrack(taskID string, times []int64) error
	UpdateDoc(taskID string, setters []DocSetter) error
	TodayItems() ([]byte, error)
}

// DocSetter represents a field update for POST /api/doc/update.
type DocSetter struct {
	Key string      `json:"key"`
	Val interface{} `json:"val"`
}

type marvinClient struct {
	httpClient      *http.Client
	token           string
	fullAccessToken string
	baseURL         string
}

func NewMarvinClient(token, fullAccessToken string) MarvinAPIClient {
	return &marvinClient{
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		token:           token,
		fullAccessToken: fullAccessToken,
		baseURL:         marvinBaseURL,
	}
}

func (mc *marvinClient) Track(taskID string, action string) error {
	payload := fmt.Sprintf(`{"taskId":"%s","action":"%s"}`, taskID, action)
	req, err := http.NewRequest(http.MethodPost, mc.baseURL+"/track", strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Token", mc.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marvin track error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marvin track returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateDoc updates fields on a task document via POST /api/doc/update.
// This updates the CouchDB document that the Marvin web client reads.
// Requires a Full Access Token.
func (mc *marvinClient) UpdateDoc(taskID string, setters []DocSetter) error {
	if mc.fullAccessToken == "" {
		return fmt.Errorf("full access token not configured")
	}

	payload := map[string]interface{}{
		"itemId":  taskID,
		"setters": setters,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("doc/update marshal error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, mc.baseURL+"/doc/update", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("X-Full-Access-Token", mc.fullAccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marvin doc/update error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marvin doc/update returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TodayItems fetches today's tasks from Marvin via GET /todayItems.
// Returns the raw JSON response bytes for passthrough to clients.
func (mc *marvinClient) TodayItems() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, mc.baseURL+"/todayItems", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Full-Access-Token", mc.fullAccessToken)

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("marvin todayItems error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("marvin todayItems read error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("marvin todayItems returned %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Retrack updates a task's times array in the Marvin document store.
// This is required after Track("STOP") because the Track endpoint only
// toggles server-side tracking state without updating the task document.
func (mc *marvinClient) Retrack(taskID string, times []int64) error {
	payload := map[string]interface{}{
		"taskId": taskID,
		"times":  times,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("retrack marshal error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, mc.baseURL+"/retrack", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Token", mc.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marvin retrack error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marvin retrack returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
