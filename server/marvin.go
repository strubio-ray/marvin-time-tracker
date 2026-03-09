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

// TrackedItemResponse represents the response from GET /api/trackedItem.
type TrackedItemResponse struct {
	TaskID  string `json:"taskId"`
	Title   string `json:"title"`
	StartedAt int64 `json:"startedAt"`
}

// MarvinAPIClient interfaces with the Marvin API.
type MarvinAPIClient interface {
	GetTrackedItem() (*TrackedItemResponse, error)
	Track(taskID string, action string) error
	Retrack(taskID string, times []int64) error
	UpdateDoc(taskID string, setters []DocSetter) error
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

func (mc *marvinClient) GetTrackedItem() (*TrackedItemResponse, error) {
	req, err := http.NewRequest(http.MethodGet, mc.baseURL+"/trackedItem", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Token", mc.token)

	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("marvin API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("marvin API returned %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Empty body means no tracked item
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return nil, nil
	}

	var item TrackedItemResponse
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("marvin API decode error: %w", err)
	}

	if item.TaskID == "" {
		return nil, nil
	}

	return &item, nil
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
