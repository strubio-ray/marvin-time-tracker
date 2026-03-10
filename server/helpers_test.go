package main

import "sync"

type mockNotifier struct {
	mu              sync.Mutex
	startCalls      int
	updateCalls     int
	endCalls        int
	silentPushCalls int
	alertPushCalls  int

	lastSilentToken string
	lastAlertToken  string
	lastAlertTitle  string
	lastAlertBody   string
}

func (m *mockNotifier) StartActivity(token, title string, startedAt int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startCalls++
	return nil
}
func (m *mockNotifier) UpdateActivity(token, title string, startedAt int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateCalls++
	return nil
}
func (m *mockNotifier) EndActivity(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.endCalls++
	return nil
}
func (m *mockNotifier) SendSilentPush(deviceToken string, taskTitle string, startedAtMs int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.silentPushCalls++
	m.lastSilentToken = deviceToken
	return nil
}
func (m *mockNotifier) SendAlertPush(deviceToken string, title string, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertPushCalls++
	m.lastAlertToken = deviceToken
	m.lastAlertTitle = title
	m.lastAlertBody = body
	return nil
}

// Thread-safe accessors for test assertions
func (m *mockNotifier) getAlertPushCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.alertPushCalls
}

func (m *mockNotifier) getAlertTitle() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastAlertTitle
}

func (m *mockNotifier) getAlertBody() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastAlertBody
}
