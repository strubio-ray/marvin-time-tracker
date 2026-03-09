package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

const (
	apnsPushType   = "liveactivity"
	attributesType = "TimeTrackerAttributes"
)

// apnsEnv is set at build time via -ldflags. Valid values: "development", "production".
var apnsEnv = "development"

type APNsClient struct {
	client   *apns2.Client
	topic    string // liveactivity topic (bundleID + ".push-type.liveactivity")
	bundleID string // bare bundle ID for regular pushes
}

func NewAPNsClient(keyPath, keyID, teamID, bundleID string) (*APNsClient, error) {
	authKey, err := token.AuthKeyFromFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load APNs key: %w", err)
	}

	tok := &token.Token{
		AuthKey: authKey,
		KeyID:   keyID,
		TeamID:  teamID,
	}

	client := apns2.NewTokenClient(tok)
	if apnsEnv == "production" {
		client = client.Production()
	} else {
		client = client.Development()
	}

	return &APNsClient{
		client:   client,
		topic:    bundleID + ".push-type.liveactivity",
		bundleID: bundleID,
	}, nil
}

func (ac *APNsClient) StartActivity(pushToStartToken string, taskTitle string, startedAtMs int64) error {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp":  time.Now().Unix(),
			"event":      "start",
			"content-state": map[string]interface{}{
				"taskTitle":  taskTitle,
				"startedAt":  float64(startedAtMs)/1000.0 - 978307200.0,
				"isTracking": true,
			},
			"attributes-type": attributesType,
			"attributes":      map[string]interface{}{},
			"alert": map[string]interface{}{
				"title": "Tracking Started",
				"body":  taskTitle,
			},
		},
	}

	return ac.send(pushToStartToken, payload, 10, ac.topic, apnsPushType)
}

func (ac *APNsClient) UpdateActivity(updateToken string, taskTitle string, startedAtMs int64) error {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"event":     "update",
			"content-state": map[string]interface{}{
				"taskTitle":  taskTitle,
				"startedAt":  float64(startedAtMs)/1000.0 - 978307200.0,
				"isTracking": true,
			},
		},
	}

	return ac.send(updateToken, payload, 10, ac.topic, apnsPushType)
}

func (ac *APNsClient) EndActivity(updateToken string) error {
	now := time.Now()
	// Apple reference date: 2001-01-01T00:00:00Z
	appleRefDate := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	startedAtApple := now.Sub(appleRefDate).Seconds()

	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp":      now.Unix(),
			"event":          "end",
			"dismissal-date": now.Unix(),
			"content-state": map[string]interface{}{
				"taskTitle":  "",
				"startedAt":  startedAtApple,
				"isTracking": false,
			},
		},
	}

	return ac.send(updateToken, payload, 10, ac.topic, apnsPushType)
}

func (ac *APNsClient) SendSilentPush(deviceToken string, taskTitle string, startedAtMs int64) error {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"content-available": 1,
		},
		"action":    "refreshStatus",
		"taskTitle": taskTitle,
		"startedAt": startedAtMs,
	}

	return ac.send(deviceToken, payload, 5, ac.bundleID, "background")
}

func (ac *APNsClient) SendAlertPush(deviceToken string, title string, body string) error {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]interface{}{
				"title": title,
				"body":  body,
			},
			"sound": "default",
		},
	}

	return ac.send(deviceToken, payload, 10, ac.bundleID, "alert")
}

func (ac *APNsClient) send(deviceToken string, payload map[string]interface{}, priority int, topic string, pushType string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       topic,
		Payload:     payloadBytes,
		Priority:    priority,
		PushType:    apns2.EPushType(pushType),
	}

	resp, err := ac.client.Push(notification)
	if err != nil {
		return fmt.Errorf("APNs push error: %w", err)
	}

	if !resp.Sent() {
		return fmt.Errorf("APNs rejected: %d %s", resp.StatusCode, resp.Reason)
	}

	log.Printf("APNs: sent %s push to %s...%s", pushType, deviceToken[:8], deviceToken[len(deviceToken)-4:])
	return nil
}

// marshalSilentPushPayload is exported for testing payload structure.
func marshalSilentPushPayload(taskTitle string, startedAtMs int64) ([]byte, error) {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"content-available": 1,
		},
		"action":    "refreshStatus",
		"taskTitle": taskTitle,
		"startedAt": startedAtMs,
	}
	return json.Marshal(payload)
}

// marshalAlertPushPayload is exported for testing payload structure.
func marshalAlertPushPayload(title string, body string) ([]byte, error) {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"alert": map[string]interface{}{
				"title": title,
				"body":  body,
			},
			"sound": "default",
		},
	}
	return json.Marshal(payload)
}

// marshalAPNsPayload is exported for testing payload structure.
func marshalAPNsPayload(event string, taskTitle string, startedAtMs int64, isTracking bool) ([]byte, error) {
	payload := map[string]interface{}{
		"aps": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"event":     event,
			"content-state": map[string]interface{}{
				"taskTitle":  taskTitle,
				"startedAt":  float64(startedAtMs)/1000.0 - 978307200.0,
				"isTracking": isTracking,
			},
		},
	}

	if event == "start" {
		aps := payload["aps"].(map[string]interface{})
		aps["attributes-type"] = attributesType
		aps["attributes"] = map[string]interface{}{}
		aps["alert"] = map[string]interface{}{
			"title": "Tracking Started",
			"body":  taskTitle,
		}
	}

	if event == "end" {
		aps := payload["aps"].(map[string]interface{})
		aps["dismissal-date"] = time.Now().Unix()
	}

	return json.Marshal(payload)
}
