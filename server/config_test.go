package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("MARVIN_API_TOKEN", "test-token")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MarvinAPIToken != "test-token" {
		t.Errorf("expected MarvinAPIToken test-token, got %s", cfg.MarvinAPIToken)
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("expected default ListenAddr :8080, got %s", cfg.ListenAddr)
	}
	if cfg.StateFilePath != "./state.json" {
		t.Errorf("expected default StateFilePath ./state.json, got %s", cfg.StateFilePath)
	}
	if cfg.PollIntervalActive != 30*time.Second {
		t.Errorf("expected default PollIntervalActive 30s, got %v", cfg.PollIntervalActive)
	}
	if cfg.PollIntervalIdle != 5*time.Minute {
		t.Errorf("expected default PollIntervalIdle 5m, got %v", cfg.PollIntervalIdle)
	}
	if cfg.APNsBundleID != "com.strubio.MarvinTimeTracker" {
		t.Errorf("expected default APNsBundleID, got %s", cfg.APNsBundleID)
	}
}

func TestLoadConfigMissingToken(t *testing.T) {
	os.Unsetenv("MARVIN_API_TOKEN")

	_, err := LoadConfig("")
	if err == nil {
		t.Fatal("expected error for missing MARVIN_API_TOKEN")
	}
}

func TestLoadConfigCustomValues(t *testing.T) {
	t.Setenv("MARVIN_API_TOKEN", "tok")
	t.Setenv("LISTEN_ADDR", ":9090")
	t.Setenv("POLL_INTERVAL_ACTIVE", "10s")
	t.Setenv("STATE_FILE_PATH", "/tmp/state.json")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ListenAddr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.ListenAddr)
	}
	if cfg.PollIntervalActive != 10*time.Second {
		t.Errorf("expected 10s, got %v", cfg.PollIntervalActive)
	}
	if cfg.StateFilePath != "/tmp/state.json" {
		t.Errorf("expected /tmp/state.json, got %s", cfg.StateFilePath)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config")
	content := `# Marvin API
MARVIN_API_TOKEN = file-token
LISTEN_ADDR = :3000
POLL_INTERVAL_ACTIVE = 15s
`
	if err := os.WriteFile(cfgFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MarvinAPIToken != "file-token" {
		t.Errorf("expected file-token, got %s", cfg.MarvinAPIToken)
	}
	if cfg.ListenAddr != ":3000" {
		t.Errorf("expected :3000, got %s", cfg.ListenAddr)
	}
	if cfg.PollIntervalActive != 15*time.Second {
		t.Errorf("expected 15s, got %v", cfg.PollIntervalActive)
	}
	// Defaults still apply for unset values
	if cfg.StateFilePath != "./state.json" {
		t.Errorf("expected default StateFilePath, got %s", cfg.StateFilePath)
	}
}

func TestLoadConfigEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config")
	content := `MARVIN_API_TOKEN = file-token
LISTEN_ADDR = :3000
`
	if err := os.WriteFile(cfgFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("LISTEN_ADDR", ":9999")

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MarvinAPIToken != "file-token" {
		t.Errorf("expected file-token, got %s", cfg.MarvinAPIToken)
	}
	// Env var should override file value
	if cfg.ListenAddr != ":9999" {
		t.Errorf("expected env override :9999, got %s", cfg.ListenAddr)
	}
}

func TestLoadConfigFileMissing(t *testing.T) {
	t.Setenv("MARVIN_API_TOKEN", "tok")

	cfg, err := LoadConfig("/nonexistent/config")
	if err != nil {
		t.Fatalf("expected missing file to be ignored, got: %v", err)
	}
	if cfg.MarvinAPIToken != "tok" {
		t.Errorf("expected tok, got %s", cfg.MarvinAPIToken)
	}
}

func TestLoadConfigFileQuotedValues(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config")
	content := `MARVIN_API_TOKEN = "quoted-token"
LISTEN_ADDR = ':4000'
`
	if err := os.WriteFile(cfgFile, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MarvinAPIToken != "quoted-token" {
		t.Errorf("expected quoted-token, got %s", cfg.MarvinAPIToken)
	}
	if cfg.ListenAddr != ":4000" {
		t.Errorf("expected :4000, got %s", cfg.ListenAddr)
	}
}
