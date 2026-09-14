package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
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
