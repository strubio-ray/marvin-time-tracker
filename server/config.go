package main

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	APNsKeyID         string
	APNsTeamID        string
	APNsPrivateKeyPath string
	APNsBundleID      string
	MarvinAPIToken    string
	StateFilePath     string
	ListenAddr        string
	PollIntervalActive time.Duration
	PollIntervalIdle   time.Duration
}

func LoadConfig() (*Config, error) {
	token := os.Getenv("MARVIN_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("MARVIN_API_TOKEN is required")
	}

	cfg := &Config{
		APNsKeyID:          os.Getenv("APNS_KEY_ID"),
		APNsTeamID:         os.Getenv("APNS_TEAM_ID"),
		APNsPrivateKeyPath: os.Getenv("APNS_KEY_P8_PATH"),
		APNsBundleID:       envOrDefault("APNS_BUNDLE_ID", "com.strubio.MarvinTimeTracker"),
		MarvinAPIToken:     token,
		StateFilePath:      envOrDefault("STATE_FILE_PATH", "./state.json"),
		ListenAddr:         envOrDefault("LISTEN_ADDR", ":8080"),
		PollIntervalActive: parseDurationOrDefault("POLL_INTERVAL_ACTIVE", 30*time.Second),
		PollIntervalIdle:   parseDurationOrDefault("POLL_INTERVAL_IDLE", 5*time.Minute),
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDurationOrDefault(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
