package main

// Notifier sends Live Activity push notifications via APNs.
type Notifier interface {
	StartActivity(token string, taskTitle string, startedAtMs int64) error
	UpdateActivity(token string, taskTitle string, startedAtMs int64) error
	EndActivity(token string) error
	SendSilentPush(deviceToken string, taskTitle string, startedAtMs int64) error
	SendAlertPush(deviceToken string, title string, body string) error
}
