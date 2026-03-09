package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	LastPollAt           time.Time `json:"lastPollAt,omitempty"`
	LastWebhookAt        time.Time `json:"lastWebhookAt,omitempty"`
	LastStopAt           time.Time `json:"lastStopAt,omitempty"`
	LastStoppedTaskID    string    `json:"lastStoppedTaskId,omitempty"`
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
	data, err := json.MarshalIndent(ss.state, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(ss.filePath)
	tmp, err := os.CreateTemp(dir, "state-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, ss.filePath)
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

func (ss *StateStore) Clear() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.state = State{}
	return ss.saveLocked()
}
