package cmd

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/session"
	"github.com/Nerver-zip/breathing-tui/internal/storage"
)

func setupTestEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
}

func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestCLIConfigCommands(t *testing.T) {
	setupTestEnv(t)

	// config path
	out, err := executeCommand("config", "path")
	if err != nil {
		t.Fatalf("config path error: %v", err)
	}
	if !strings.Contains(out, "config.yaml") {
		t.Fatalf("unexpected path output: %s", out)
	}

	// config show
	out, err = executeCommand("config", "show")
	if err != nil {
		t.Fatalf("config show error: %v", err)
	}
	if !strings.Contains(out, "rounds: 3") || !strings.Contains(out, "theme: default") {
		t.Fatalf("unexpected show output: %s", out)
	}

	// config set rounds 4
	out, err = executeCommand("config", "set", "rounds", "4")
	if err != nil {
		t.Fatalf("config set error: %v", err)
	}
	if !strings.Contains(out, "Updated rounds to 4") {
		t.Fatalf("unexpected set output: %s", out)
	}

	// verify via config show
	out, err = executeCommand("config", "show")
	if err != nil {
		t.Fatalf("config show error: %v", err)
	}
	if !strings.Contains(out, "rounds: 4") {
		t.Fatalf("expected rounds: 4, got: %s", out)
	}
}

func TestCLIThemeCommands(t *testing.T) {
	setupTestEnv(t)

	// theme list
	out, err := executeCommand("theme", "list")
	if err != nil {
		t.Fatalf("theme list error: %v", err)
	}
	if !strings.Contains(out, "dracula") || !strings.Contains(out, "catppuccin-mocha") {
		t.Fatalf("unexpected theme list: %s", out)
	}

	// theme preview
	out, err = executeCommand("theme", "preview", "dracula")
	if err != nil {
		t.Fatalf("theme preview error: %v", err)
	}
	if !strings.Contains(out, "dracula") || !strings.Contains(out, "Color Swatches") {
		t.Fatalf("unexpected theme preview: %s", out)
	}

	// theme set
	out, err = executeCommand("theme", "set", "gruvbox")
	if err != nil {
		t.Fatalf("theme set error: %v", err)
	}
	if !strings.Contains(out, "Theme changed to 'gruvbox'") {
		t.Fatalf("unexpected theme set output: %s", out)
	}
}

func TestCLIStatsPlainAndJSON(t *testing.T) {
	setupTestEnv(t)

	// stats --plain
	out, err := executeCommand("stats", "--plain")
	if err != nil {
		t.Fatalf("stats --plain error: %v", err)
	}
	if !strings.Contains(out, "Today:") || !strings.Contains(out, "Retention:") || !strings.Contains(out, "Streak:") {
		t.Fatalf("unexpected stats --plain output: %s", out)
	}

	// stats --json
	out, err = executeCommand("stats", "--json")
	if err != nil {
		t.Fatalf("stats --json error: %v", err)
	}
	if !strings.Contains(out, `"today_sessions"`) || !strings.Contains(out, `"current_streak"`) {
		t.Fatalf("unexpected stats --json output: %s", out)
	}
}

func TestCLIStartFlagsHelp(t *testing.T) {
	setupTestEnv(t)

	out, err := executeCommand("start", "--help")
	if err != nil {
		t.Fatalf("start --help error: %v", err)
	}
	if !strings.Contains(out, "--counted") || !strings.Contains(out, "--timed") || !strings.Contains(out, "--breaths") {
		t.Fatalf("expected flags --counted, --timed, --breaths in help output, got:\n%s", out)
	}
}

func TestCLIVersionFlag(t *testing.T) {
	out, err := executeCommand("--version")
	if err != nil {
		t.Fatalf("version command error: %v", err)
	}
	if strings.TrimSpace(out) != "breath version dev" {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestCLICleanSessions(t *testing.T) {
	setupTestEnv(t)

	// Clean when empty
	out, err := executeCommand("stats", "clean")
	if err == nil || !strings.Contains(err.Error(), "no sessions found") {
		t.Fatalf("expected error cleaning empty store, got out=%q, err=%v", out, err)
	}

	// Create 3 sessions
	store, err := storage.Open("")
	if err != nil {
		t.Fatalf("storage.Open error: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	t1 := time.Now().Add(-2 * time.Hour)
	s1, _ := store.CreateSession(ctx, 3, t1)
	_ = store.SaveRound(ctx, s1, session.RoundResult{Index: 1, Retention: 30 * time.Second})
	_ = store.EndSession(ctx, s1, t1.Add(10*time.Minute), 10*time.Minute, 10*time.Minute, "completed")

	t2 := time.Now().Add(-1 * time.Hour)
	s2, _ := store.CreateSession(ctx, 3, t2)
	_ = store.SaveRound(ctx, s2, session.RoundResult{Index: 1, Retention: 40 * time.Second})
	_ = store.EndSession(ctx, s2, t2.Add(10*time.Minute), 10*time.Minute, 10*time.Minute, "completed")

	t3 := time.Now()
	s3, _ := store.CreateSession(ctx, 3, t3)
	_ = store.SaveRound(ctx, s3, session.RoundResult{Index: 1, Retention: 50 * time.Second})
	_ = store.EndSession(ctx, s3, t3.Add(10*time.Minute), 10*time.Minute, 10*time.Minute, "completed")

	// Delete ~1 (the past session, which is s2)
	out, err = executeCommand("stats", "clean", "~1")
	if err != nil {
		t.Fatalf("clean ~1 error: %v", err)
	}
	if !strings.Contains(out, "previous session (~1)") || !strings.Contains(out, "Deleted") {
		t.Fatalf("unexpected output deleting ~1: %s", out)
	}

	// Delete latest session (s3) via root alias "breath clean"
	out, err = executeCommand("clean")
	if err != nil {
		t.Fatalf("clean latest error: %v", err)
	}
	if !strings.Contains(out, "latest session") || !strings.Contains(out, "Deleted") {
		t.Fatalf("unexpected output deleting latest: %s", out)
	}

	// Now only s1 remains. Test clean --all
	out, err = executeCommand("stats", "clean", "--all")
	if err != nil {
		t.Fatalf("clean --all error: %v", err)
	}
	if !strings.Contains(out, "Cleared 1 session") {
		t.Fatalf("unexpected output for clean --all: %s", out)
	}

	// Running clean --all again says history is empty
	out, err = executeCommand("clean", "--all")
	if err != nil {
		t.Fatalf("clean --all second time error: %v", err)
	}
	if !strings.Contains(out, "already empty") {
		t.Fatalf("expected already empty message, got: %s", out)
	}
}
