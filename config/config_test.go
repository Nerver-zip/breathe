package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.Rounds != 3 || d.Breathing != 3*time.Minute || d.Recovery != 30*time.Second || d.AutoNextRound != false || d.Theme != "default" || !d.Notifications || !d.Bell {
		t.Fatalf("unexpected defaults: %#v", d)
	}
}

func TestLoadCreatesDefaultFile(t *testing.T) {
	dir := setupTestConfig(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Rounds != 3 {
		t.Fatalf("expected 3 rounds, got %d", cfg.Rounds)
	}

	expectedFile := filepath.Join(dir, "breath", "config.yaml")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected config file %s to be created", expectedFile)
	}
}

func TestSetValidKeys(t *testing.T) {
	setupTestConfig(t)

	cfg, err := Set("rounds", "5")
	if err != nil || cfg.Rounds != 5 {
		t.Fatalf("Set rounds failed: %v, %#v", err, cfg)
	}

	cfg, err = Set("breathing", "2m30s")
	if err != nil || cfg.Breathing != 2*time.Minute+30*time.Second {
		t.Fatalf("Set breathing failed: %v", err)
	}

	cfg, err = Set("recovery", "45s")
	if err != nil || cfg.Recovery != 45*time.Second {
		t.Fatalf("Set recovery failed: %v", err)
	}

	cfg, err = Set("auto_next_round", "true")
	if err != nil || !cfg.AutoNextRound {
		t.Fatalf("Set auto_next_round failed: %v", err)
	}

	cfg, err = Set("theme", "nord")
	if err != nil || cfg.Theme != "nord" {
		t.Fatalf("Set theme failed: %v", err)
	}

	cfg, err = Set("notifications", "false")
	if err != nil || cfg.Notifications {
		t.Fatalf("Set notifications failed: %v", err)
	}

	cfg, err = Set("bell", "false")
	if err != nil || cfg.Bell {
		t.Fatalf("Set bell failed: %v", err)
	}

	cfg, err = Set("font", "mono12")
	if err != nil || cfg.Font != "mono12" {
		t.Fatalf("Set font failed: %v", err)
	}

	cfg, err = Set("mode", "counted")
	if err != nil || cfg.Mode != "counted" {
		t.Fatalf("Set mode failed: %v", err)
	}

	cfg, err = Set("breaths", "40")
	if err != nil || cfg.Breaths != 40 {
		t.Fatalf("Set breaths failed: %v", err)
	}

	cfg, err = Set("quote_interval", "15s")
	if err != nil || cfg.QuoteInterval != 15*time.Second {
		t.Fatalf("Set quote_interval failed: %v", err)
	}

	cfg, err = Set("quotes", "Breathe in, breathe out; Stay calm")
	if err != nil || len(cfg.Quotes) != 2 || cfg.Quotes[0] != "Breathe in, breathe out" || cfg.Quotes[1] != "Stay calm" {
		t.Fatalf("Set quotes failed: %v, %#v", err, cfg.Quotes)
	}

	// Verify persistence by loading afresh
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Rounds != 5 || loaded.Theme != "nord" || loaded.Notifications || loaded.Bell || loaded.Font != "mono12" || loaded.Mode != "counted" || loaded.Breaths != 40 || loaded.QuoteInterval != 15*time.Second || len(loaded.Quotes) != 2 {
		t.Fatalf("re-loaded config mismatch: %#v", loaded)
	}
}

func TestSetInvalidKeysAndValues(t *testing.T) {
	setupTestConfig(t)

	if _, err := Set("unknown_key", "val"); err == nil || !strings.Contains(err.Error(), "Supported keys:") {
		t.Fatalf("expected unknown key error, got: %v", err)
	}

	if _, err := Set("rounds", "0"); err == nil {
		t.Fatal("expected error for rounds 0")
	}

	if _, err := Set("breathing", "not-a-duration"); err == nil {
		t.Fatal("expected error for invalid breathing duration")
	}

	if _, err := Set("theme", "made-up-theme"); err == nil {
		t.Fatal("expected error for invalid theme")
	}

	if _, err := Set("bell", "not-a-bool"); err == nil {
		t.Fatal("expected error for invalid bell boolean")
	}

	if _, err := Set("font", "non-existent-font"); err == nil {
		t.Fatal("expected error for invalid font")
	}
}
