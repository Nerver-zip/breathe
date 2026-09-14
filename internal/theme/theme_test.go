package theme

import (
	"strings"
	"testing"
)

func TestBuiltInThemes(t *testing.T) {
	required := []string{
		"default",
		"pomo",
		"catppuccin-mocha",
		"dracula",
		"gruvbox",
		"nord",
		"tokyo-night",
		"solarized",
	}

	for _, name := range required {
		th, err := Get(name)
		if err != nil {
			t.Errorf("Get(%q) returned error: %v", name, err)
		}
		if th.Name != name {
			t.Errorf("expected Name %q, got %q", name, th.Name)
		}
	}
}

func TestGetCaseInsensitive(t *testing.T) {
	th, err := Get("Catppuccin-Mocha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if th.Name != "catppuccin-mocha" {
		t.Fatalf("expected catppuccin-mocha, got %s", th.Name)
	}
}

func TestUnknownThemeError(t *testing.T) {
	_, err := Get("non-existent-theme")
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
	if !strings.Contains(err.Error(), "Available themes:") {
		t.Fatalf("expected actionable error with available themes, got %v", err)
	}
}

func TestPreview(t *testing.T) {
	out, err := Preview("dracula")
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}
	if !strings.Contains(out, "dracula") || !strings.Contains(out, "Color Swatches") {
		t.Fatalf("unexpected preview output: %s", out)
	}
}
