package main

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type State struct {
	TrackingTaskID       string    `json:"trackingTaskId,omitempty"`
	TaskTitle            string    `json:"taskTitle,omitempty"`
	StartedAt            int64     `json:"startedAt,omitempty"`
	Times                []int64   `json:"times,omitempty"`
	PushToStartToken     string    `json:"pushToStartToken,omitempty"`
	UpdateToken          string    `json:"updateToken,omitempty"`
	DeviceToken          string    `json:"deviceToken,omitempty"`
	LiveActivityStartedAt time.Time `json:"liveActivityStartedAt,omitempty"`
	LastWebhookAt        time.Time `json:"lastWebhookAt,omitempty"`
	LastStopAt           time.Time `json:"lastStopAt,omitempty"`
}

func (s State) IsTracking() bool {
	return s.TrackingTaskID != ""
}

type StateStore struct {
	mu       sync.Mutex
	filePath string
	state    State
}

func NewStateStore(filePath string) *StateStore {
	return &StateStore{filePath: filePath}
}

func (ss *StateStore) Load() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	data, err := os.ReadFile(ss.filePath)
	if os.IsNotExist(err) {
		ss.state = State{}
		return nil
	}
	if err != nil {
		return err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ss.state = s
	return nil
}

func (ss *StateStore) Save() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.saveLocked()
}

func (ss *StateStore) saveLocked() error {
	return atomicWriteJSON(ss.filePath, ss.state)
}

func (ss *StateStore) Get() State {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.state
}

func (ss *StateStore) Update(fn func(*State)) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	fn(&ss.state)
	return ss.saveLocked()
}

// ClearTracking zeroes tracking fields and sets LastStopAt.
// Returns the state snapshot taken before clearing, for callers that need
// the previous update token or task ID.
// Optional extra mutators are applied after clearing (e.g. to set LastWebhookAt).
func (ss *StateStore) ClearTracking(now time.Time, extras ...func(*State)) (State, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	prev := ss.state
	ss.state.TrackingTaskID = ""
	ss.state.TaskTitle = ""
	ss.state.StartedAt = 0
	ss.state.Times = nil
	ss.state.LastStopAt = now
	ss.state.LiveActivityStartedAt = time.Time{}
	ss.state.UpdateToken = ""
	for _, fn := range extras {
		fn(&ss.state)
	}
	return prev, ss.saveLocked()
}

// ConsumeNotifyTokens returns the current push tokens and atomically clears
// the single-use PushToStartToken if present.
func (ss *StateStore) ConsumeNotifyTokens() (NotifyTokens, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	tokens := NotifyTokens{
		UpdateToken:      ss.state.UpdateToken,
		PushToStartToken: ss.state.PushToStartToken,
		DeviceToken:      ss.state.DeviceToken,
	}
	if ss.state.PushToStartToken != "" {
		ss.state.PushToStartToken = ""
		if err := ss.saveLocked(); err != nil {
			return tokens, err
		}
	}
	return tokens, nil
}

func (ss *StateStore) Clear() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.state = State{}
	return ss.saveLocked()
}
