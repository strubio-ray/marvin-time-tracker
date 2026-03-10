package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/http2"
)

const (
	apnsPushType   = "liveactivity"
	attributesType = "TimeTrackerAttributes"

	apnsHostDevelopment = "https://api.sandbox.push.apple.com"
	apnsHostProduction  = "https://api.push.apple.com"

	// Apple epoch: 2001-01-01T00:00:00Z in Unix seconds
	appleEpochUnixSec int64 = 978307200

	// JWT tokens are cached for 50 minutes (Apple allows up to 60).
	jwtCacheDuration = 50 * time.Minute

	// Retry configuration
	maxRetries     = 3
	retryBaseDelay = 1 * time.Second
)

// msToAppleSeconds converts a Unix millisecond timestamp to Apple epoch seconds.
func msToAppleSeconds(ms int64) float64 {
	return float64(ms)/1000.0 - float64(appleEpochUnixSec)
}

type APNsClient struct {
	httpClient *http.Client
	keyID      string
	teamID     string
	privateKey *ecdsa.PrivateKey
	bundleID   string
	host       string

	mu        sync.Mutex
	cachedJWT string
	jwtExpiry time.Time
}

func NewAPNsClient(keyPath, keyID, teamID, bundleID, env string) (*APNsClient, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read APNs key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from APNs key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse APNs private key: %w", err)
	}

	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("APNs key is not an ECDSA key")
	}

	host := apnsHostDevelopment
	if env == "production" {
		host = apnsHostProduction
	}

	transport := &http.Transport{}
	if err := http2.ConfigureTransport(transport); err != nil {
		return nil, fmt.Errorf("failed to configure HTTP/2 transport: %w", err)
	}

	return &APNsClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		keyID:      keyID,
		teamID:     teamID,
		privateKey: ecKey,
		bundleID:   bundleID,
		host:       host,
	}, nil
}

func (ac *APNsClient) getJWT() (string, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.cachedJWT != "" && time.Now().Before(ac.jwtExpiry) {
		return ac.cachedJWT, nil
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": ac.teamID,
		"iat": now.Unix(),
	})
	token.Header["kid"] = ac.keyID

	signed, err := token.SignedString(ac.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	ac.cachedJWT = signed
	ac.jwtExpiry = now.Add(jwtCacheDuration)
	return signed, nil
}

// buildContentState creates the content-state dictionary for Live Activity payloads.
func buildContentState(taskTitle string, startedAtMs int64, isTracking bool) map[string]interface{} {
	return map[string]interface{}{
		"taskTitle":  taskTitle,
		"startedAt":  msToAppleSeconds(startedAtMs),
		"isTracking": isTracking,
	}
}

// buildActivityPayload creates a Live Activity APNs payload for start/update/end events.
func buildActivityPayload(event string, taskTitle string, startedAtMs int64, isTracking bool) map[string]interface{} {
	aps := map[string]interface{}{
		"timestamp":     time.Now().Unix(),
		"event":         event,
		"content-state": buildContentState(taskTitle, startedAtMs, isTracking),
	}

	if event == "start" {
		aps["attributes-type"] = attributesType
		aps["attributes"] = map[string]interface{}{}
		aps["alert"] = map[string]interface{}{
			"title": "Tracking Started",
			"body":  taskTitle,
		}
	}

	if event == "end" {
		aps["dismissal-date"] = time.Now().Unix()
	}

	return map[string]interface{}{"aps": aps}
}

// buildSilentPayload creates a background push payload.
func buildSilentPayload(taskTitle string, startedAtMs int64) map[string]interface{} {
	return map[string]interface{}{
		"aps": map[string]interface{}{
			"content-available": 1,
		},
		"action":    "refreshStatus",
		"taskTitle": taskTitle,
		"startedAt": startedAtMs,
	}
}

// buildAlertPayload creates an alert push payload.
func buildAlertPayload(title string, body string) map[string]interface{} {
	return map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]interface{}{
				"title": title,
				"body":  body,
			},
			"sound": "default",
		},
	}
}

func (ac *APNsClient) StartActivity(pushToStartToken string, taskTitle string, startedAtMs int64) error {
	payload := buildActivityPayload("start", taskTitle, startedAtMs, true)
	return ac.send(pushToStartToken, payload, 10, ac.topic(), apnsPushType)
}

func (ac *APNsClient) UpdateActivity(updateToken string, taskTitle string, startedAtMs int64) error {
	payload := buildActivityPayload("update", taskTitle, startedAtMs, true)
	return ac.send(updateToken, payload, 10, ac.topic(), apnsPushType)
}

func (ac *APNsClient) EndActivity(updateToken string) error {
	now := time.Now()
	startedAtMs := now.UnixMilli()
	payload := buildActivityPayload("end", "", startedAtMs, false)
	return ac.send(updateToken, payload, 10, ac.topic(), apnsPushType)
}

func (ac *APNsClient) SendSilentPush(deviceToken string, taskTitle string, startedAtMs int64) error {
	payload := buildSilentPayload(taskTitle, startedAtMs)
	return ac.send(deviceToken, payload, 5, ac.bundleID, "background")
}

func (ac *APNsClient) SendAlertPush(deviceToken string, title string, body string) error {
	payload := buildAlertPayload(title, body)
	return ac.send(deviceToken, payload, 10, ac.bundleID, "alert")
}

func (ac *APNsClient) topic() string {
	return ac.bundleID + ".push-type.liveactivity"
}

// isRetryable returns true if the HTTP status code indicates a retryable error.
func isRetryable(statusCode int) bool {
	return statusCode >= 500
}

func (ac *APNsClient) send(deviceToken string, payload map[string]interface{}, priority int, topic string, pushType string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/3/device/%s", ac.host, deviceToken)

	var lastErr error
	for attempt := range maxRetries {
		jwtToken, err := ac.getJWT()
		if err != nil {
			return err
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payloadBytes))
		if err != nil {
			return fmt.Errorf("failed to create APNs request: %w", err)
		}

		req.Header.Set("Authorization", "bearer "+jwtToken)
		req.Header.Set("apns-topic", topic)
		req.Header.Set("apns-push-type", pushType)
		req.Header.Set("apns-priority", fmt.Sprintf("%d", priority))

		resp, err := ac.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("APNs push error: %w", err)
			if attempt < maxRetries-1 {
				time.Sleep(retryBaseDelay * time.Duration(1<<attempt))
			}
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			tokenPreview := deviceToken
			if len(deviceToken) > 12 {
				tokenPreview = deviceToken[:8] + "..." + deviceToken[len(deviceToken)-4:]
			}
			log.Printf("APNs: sent %s push to %s", pushType, tokenPreview)
			return nil
		}

		if isRetryable(resp.StatusCode) {
			lastErr = fmt.Errorf("APNs server error: %d %s", resp.StatusCode, string(respBody))
			if attempt < maxRetries-1 {
				time.Sleep(retryBaseDelay * time.Duration(1<<attempt))
			}
			continue
		}

		// 4xx errors are permanent — don't retry
		return fmt.Errorf("APNs rejected: %d %s", resp.StatusCode, string(respBody))
	}

	return lastErr
}

// marshalSilentPushPayload is exported for testing payload structure.
func marshalSilentPushPayload(taskTitle string, startedAtMs int64) ([]byte, error) {
	return json.Marshal(buildSilentPayload(taskTitle, startedAtMs))
}

// marshalAlertPushPayload is exported for testing payload structure.
func marshalAlertPushPayload(title string, body string) ([]byte, error) {
	return json.Marshal(buildAlertPayload(title, body))
}

// marshalAPNsPayload is exported for testing payload structure.
func marshalAPNsPayload(event string, taskTitle string, startedAtMs int64, isTracking bool) ([]byte, error) {
	return json.Marshal(buildActivityPayload(event, taskTitle, startedAtMs, isTracking))
}
