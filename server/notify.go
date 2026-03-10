package main

import (
	"context"
	"log"
	"time"
)

const DefaultSilentPushGracePeriod = 10 * time.Second

// NotifyTokens holds the push token set needed for notification delivery.
type NotifyTokens struct {
	UpdateToken      string
	PushToStartToken string
	DeviceToken      string
}

// notifyTrackingStarted sends push notifications for a tracking start event.
// It sends a Live Activity push (update or push-to-start) if a token is available,
// AND always sends a silent push to sync the app UI state.
// If no Live Activity token exists, a delayed alert fallback fires after gracePeriod.
// The tokenChecker is called after the grace period to check if the silent push
// succeeded (i.e., the app registered an UpdateToken).
func notifyTrackingStarted(ctx context.Context, tokens NotifyTokens, notifier Notifier, broker BrokerPublisher, taskTitle string, startedAtMs int64, gracePeriod time.Duration, tokenChecker func() string) {
	if broker != nil {
		broker.BroadcastJSON("tracking_started", map[string]interface{}{
			"taskTitle": taskTitle,
			"startedAt": startedAtMs,
		})
	}

	if notifier == nil {
		return
	}

	liveActivitySent := false

	// Live Activity push (best available token)
	if tokens.UpdateToken != "" {
		if err := notifier.UpdateActivity(tokens.UpdateToken, taskTitle, startedAtMs); err != nil {
			log.Printf("notify: update activity error: %v", err)
		} else {
			liveActivitySent = true
		}
	} else if tokens.PushToStartToken != "" {
		if err := notifier.StartActivity(tokens.PushToStartToken, taskTitle, startedAtMs); err != nil {
			log.Printf("notify: start activity error: %v", err)
		} else {
			liveActivitySent = true
		}
	}

	// Always send silent push to sync app UI
	if tokens.DeviceToken != "" {
		if err := notifier.SendSilentPush(tokens.DeviceToken, taskTitle, startedAtMs); err != nil {
			log.Printf("notify: silent push error: %v", err)
		} else {
			log.Printf("notify: sent silent push")
		}

		// Alert fallback only when no Live Activity was sent (silent push is sole channel)
		if !liveActivitySent && tokenChecker != nil {
			go func() {
				timer := time.NewTimer(gracePeriod)
				defer timer.Stop()
				select {
				case <-timer.C:
					if tokenChecker() != "" {
						log.Printf("notify: silent push succeeded, skipping alert")
						return
					}
					log.Printf("notify: alert push for %s", taskTitle)
					if err := notifier.SendAlertPush(tokens.DeviceToken, "Tracking Started", taskTitle); err != nil {
						log.Printf("notify: alert push error: %v", err)
					}
				case <-ctx.Done():
					return
				}
			}()
		}
		return
	}

	if !liveActivitySent {
		log.Printf("notify: no push tokens available")
	}
}

// notifyTrackingStopped sends push notifications for a tracking stop event.
// It ends the Live Activity if an update token is provided,
// AND sends a silent push to sync the app UI state.
func notifyTrackingStopped(notifier Notifier, broker BrokerPublisher, updateToken string, deviceToken string, stoppedTaskID string) {
	if broker != nil {
		broker.BroadcastJSON("tracking_stopped", map[string]interface{}{
			"taskId": stoppedTaskID,
		})
	}

	if notifier == nil {
		return
	}

	if updateToken != "" {
		if err := notifier.EndActivity(updateToken); err != nil {
			log.Printf("notify: end activity error: %v", err)
		}
	}

	if deviceToken != "" {
		if err := notifier.SendSilentPush(deviceToken, "", 0); err != nil {
			log.Printf("notify: silent push (stop) error: %v", err)
		} else {
			log.Printf("notify: sent silent push for stop")
		}
	}
}
