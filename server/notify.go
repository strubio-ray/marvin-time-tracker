package main

import (
	"context"
	"log"
	"time"
)

const DefaultSilentPushGracePeriod = 10 * time.Second

// notifyTrackingStarted sends push notifications for a tracking start event.
// It sends a Live Activity push (update or push-to-start) if a token is available,
// AND always sends a silent push to sync the app UI state.
// If no Live Activity token exists, a delayed alert fallback fires after gracePeriod.
func notifyTrackingStarted(ctx context.Context, store *StateStore, notifier Notifier, taskTitle string, startedAtMs int64, gracePeriod time.Duration) {
	if notifier == nil {
		return
	}

	state := store.Get()
	liveActivitySent := false

	// Live Activity push (best available token)
	if state.UpdateToken != "" {
		if err := notifier.UpdateActivity(state.UpdateToken, taskTitle, startedAtMs); err != nil {
			log.Printf("notify: update activity error: %v", err)
		} else {
			liveActivitySent = true
		}
	} else if state.PushToStartToken != "" {
		if err := notifier.StartActivity(state.PushToStartToken, taskTitle, startedAtMs); err != nil {
			log.Printf("notify: start activity error: %v", err)
		} else {
			liveActivitySent = true
		}
		// Push-to-start tokens are single-use; clear to avoid reusing a consumed token.
		store.Update(func(s *State) {
			s.PushToStartToken = ""
		})
	}

	// Always send silent push to sync app UI
	if state.DeviceToken != "" {
		if err := notifier.SendSilentPush(state.DeviceToken, taskTitle, startedAtMs); err != nil {
			log.Printf("notify: silent push error: %v", err)
		} else {
			log.Printf("notify: sent silent push")
		}

		// Alert fallback only when no Live Activity was sent (silent push is sole channel)
		if !liveActivitySent {
			go func() {
				timer := time.NewTimer(gracePeriod)
				defer timer.Stop()
				select {
				case <-timer.C:
					current := store.Get()
					if current.UpdateToken != "" {
						log.Printf("notify: silent push succeeded, skipping alert")
						return
					}
					if !current.IsTracking() {
						return
					}
					log.Printf("notify: alert push for %s", taskTitle)
					if err := notifier.SendAlertPush(current.DeviceToken, "Tracking Started", taskTitle); err != nil {
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
func notifyTrackingStopped(store *StateStore, notifier Notifier, updateToken string) {
	if notifier == nil {
		return
	}

	if updateToken != "" {
		if err := notifier.EndActivity(updateToken); err != nil {
			log.Printf("notify: end activity error: %v", err)
		}
	}

	state := store.Get()
	if state.DeviceToken != "" {
		if err := notifier.SendSilentPush(state.DeviceToken, "", 0); err != nil {
			log.Printf("notify: silent push (stop) error: %v", err)
		} else {
			log.Printf("notify: sent silent push for stop")
		}
	}
}
