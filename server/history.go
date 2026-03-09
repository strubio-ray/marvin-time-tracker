package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type SessionRecord struct {
	TaskID    string `json:"taskId"`
	Title     string `json:"title"`
	StartedAt int64  `json:"startedAt"`
	StoppedAt int64  `json:"stoppedAt"`
	Duration  int64  `json:"duration"`
}

type HistoryData struct {
	Sessions []SessionRecord `json:"sessions"`
}

type HistoryStore struct {
	mu       sync.Mutex
	filePath string
	data     HistoryData
}

func NewHistoryStore(filePath string) *HistoryStore {
	return &HistoryStore{filePath: filePath}
}

func (hs *HistoryStore) Load() error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	data, err := os.ReadFile(hs.filePath)
	if os.IsNotExist(err) {
		hs.data = HistoryData{}
		return nil
	}
	if err != nil {
		return err
	}

	var d HistoryData
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	hs.data = d
	return nil
}

func (hs *HistoryStore) saveLocked() error {
	data, err := json.MarshalIndent(hs.data, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(hs.filePath)
	tmp, err := os.CreateTemp(dir, "history-*.json")
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

	return os.Rename(tmpName, hs.filePath)
}

func (hs *HistoryStore) Add(record SessionRecord) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	hs.data.Sessions = append([]SessionRecord{record}, hs.data.Sessions...)

	if len(hs.data.Sessions) > 200 {
		hs.data.Sessions = hs.data.Sessions[:200]
	}

	return hs.saveLocked()
}

func (hs *HistoryStore) Recent(n int) []SessionRecord {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	if n > len(hs.data.Sessions) {
		n = len(hs.data.Sessions)
	}

	result := make([]SessionRecord, n)
	copy(result, hs.data.Sessions[:n])
	return result
}
