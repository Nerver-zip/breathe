package tui

import (
	"strings"
	"testing"
)

func TestRenderClockFonts(t *testing.T) {
	timeStr := "03:45"
	fontsToTest := []string{
		FontAnsiShadow,
		FontMono12,
		FontAnsi,
		FontRebel,
	}

	for _, fontName := range fontsToTest {
		rendered := RenderClock(timeStr, fontName)
		if rendered == "" {
			t.Errorf("font %s rendered empty clock", fontName)
		}
		lines := strings.Split(rendered, "\n")
		if len(lines) < 4 {
			t.Errorf("font %s rendered too few lines: %d", fontName, len(lines))
		}
	}
}

func TestRenderClockDefault(t *testing.T) {
	// Unknown font falls back to DefaultFont (ansiShadow)
	renderedUnknown := RenderClock("12:34", "nonexistent")
	renderedDefault := RenderClock("12:34", FontAnsiShadow)
	if renderedUnknown != renderedDefault {
		t.Errorf("expected fallback to ansiShadow, got different rendering")
	}
}
