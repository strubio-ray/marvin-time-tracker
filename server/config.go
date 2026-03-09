package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	APNsKeyID          string
	APNsTeamID         string
	APNsPrivateKeyPath string
	APNsBundleID       string
	MarvinAPIToken     string
	StateFilePath      string
	ListenAddr         string
	PollEnabled        bool
	PollIntervalActive time.Duration
	PollIntervalIdle   time.Duration
}

// LoadConfig builds a Config by layering (lowest to highest priority):
// config file defaults → environment variables → hardcoded defaults.
func LoadConfig(configPath string) (*Config, error) {
	fileVals, err := loadConfigFile(configPath)
	if err != nil {
		return nil, err
	}

	get := func(key string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fileVals[key]
	}

	token := get("MARVIN_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("MARVIN_API_TOKEN is required")
	}

	cfg := &Config{
		APNsKeyID:          get("APNS_KEY_ID"),
		APNsTeamID:         get("APNS_TEAM_ID"),
		APNsPrivateKeyPath: expandHome(get("APNS_KEY_P8_PATH")),
		APNsBundleID:       getOrDefault(get, "APNS_BUNDLE_ID", "com.strubio.MarvinTimeTracker"),
		MarvinAPIToken:     token,
		StateFilePath:      getOrDefault(get, "STATE_FILE_PATH", "./state.json"),
		ListenAddr:         getOrDefault(get, "LISTEN_ADDR", ":8080"),
		PollEnabled:        getOrDefault(get, "POLL_ENABLED", "true") == "true",
		PollIntervalActive: parseDuration(get("POLL_INTERVAL_ACTIVE"), 30*time.Second),
		PollIntervalIdle:   parseDuration(get("POLL_INTERVAL_IDLE"), 5*time.Minute),
	}

	return cfg, nil
}

func getOrDefault(get func(string) string, key, fallback string) string {
	if v := get(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(v string, fallback time.Duration) time.Duration {
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + path[1:]
		}
	}
	return path
}

// loadConfigFile parses a KEY=value config file. Blank lines and lines
// starting with # are ignored. Returns an empty map if path is empty or
// the file does not exist.
func loadConfigFile(path string) (map[string]string, error) {
	vals := make(map[string]string)
	if path == "" {
		return vals, nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vals, nil
		}
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key != "" {
			vals[key] = val
		}
	}
	return vals, scanner.Err()
}
