package theme

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Primary       lipgloss.Color `json:"primary"`
	Secondary     lipgloss.Color `json:"secondary"`
	Accent        lipgloss.Color `json:"accent"`
	Muted         lipgloss.Color `json:"muted"`
	Good          lipgloss.Color `json:"good"`
	Warn          lipgloss.Color `json:"warn"`
	Error         lipgloss.Color `json:"error"`
	GradientStart lipgloss.Color `json:"gradient_start,omitempty"`
	GradientEnd   lipgloss.Color `json:"gradient_end,omitempty"`
}

var registry = map[string]Theme{
	"default": {
		Name:          "default",
		Description:   "Clean neutral slate with sky-blue accents",
		Primary:       lipgloss.Color("#E5E7EB"),
		Secondary:     lipgloss.Color("#9CA3AF"),
		Accent:        lipgloss.Color("#60A5FA"),
		Muted:         lipgloss.Color("#6B7280"),
		Good:          lipgloss.Color("#34D399"),
		Warn:          lipgloss.Color("#FBBF24"),
		Error:         lipgloss.Color("#F87171"),
		GradientStart: lipgloss.Color("#60A5FA"),
		GradientEnd:   lipgloss.Color("#A78BFA"),
	},
	"pomo": {
		Name:          "pomo",
		Description:   "Pomo-inspired signature indigo palette (#5A56E0)",
		Primary:       lipgloss.Color("#5A56E0"),
		Secondary:     lipgloss.Color("#8860FF"),
		Accent:        lipgloss.Color("#5A56E0"),
		Muted:         lipgloss.Color("#606060"),
		Good:          lipgloss.Color("#198754"),
		Warn:          lipgloss.Color("#F25D94"),
		Error:         lipgloss.Color("#FF4C4C"),
		GradientStart: lipgloss.Color("#5A56E0"),
		GradientEnd:   lipgloss.Color("#EE6FF8"),
	},
	"catppuccin-mocha": {
		Name:          "catppuccin-mocha",
		Description:   "Soothing pastel palette with mauve accent",
		Primary:       lipgloss.Color("#CDD6F4"),
		Secondary:     lipgloss.Color("#BAC2DE"),
		Accent:        lipgloss.Color("#CBA6F7"),
		Muted:         lipgloss.Color("#6C7086"),
		Good:          lipgloss.Color("#A6E3A1"),
		Warn:          lipgloss.Color("#F9E2AF"),
		Error:         lipgloss.Color("#F38BA8"),
		GradientStart: lipgloss.Color("#89B4FA"),
		GradientEnd:   lipgloss.Color("#CBA6F7"),
	},
	"dracula": {
		Name:          "dracula",
		Description:   "Vibrant dark theme with purple and pink highlights",
		Primary:       lipgloss.Color("#F8F8F2"),
		Secondary:     lipgloss.Color("#BD93F9"),
		Accent:        lipgloss.Color("#FF79C6"),
		Muted:         lipgloss.Color("#6272A4"),
		Good:          lipgloss.Color("#50FA7B"),
		Warn:          lipgloss.Color("#F1FA8C"),
		Error:         lipgloss.Color("#FF5555"),
		GradientStart: lipgloss.Color("#BD93F9"),
		GradientEnd:   lipgloss.Color("#FF79C6"),
	},
	"gruvbox": {
		Name:          "gruvbox",
		Description:   "Warm retro groove with autumn orange and olive",
		Primary:       lipgloss.Color("#EBDBB2"),
		Secondary:     lipgloss.Color("#D5C4A1"),
		Accent:        lipgloss.Color("#FE8019"),
		Muted:         lipgloss.Color("#928374"),
		Good:          lipgloss.Color("#B8BB26"),
		Warn:          lipgloss.Color("#FABD2F"),
		Error:         lipgloss.Color("#FB4934"),
		GradientStart: lipgloss.Color("#FE8019"),
		GradientEnd:   lipgloss.Color("#FABD2F"),
	},
	"nord": {
		Name:          "nord",
		Description:   "Arctic, north-bluish clean aesthetic",
		Primary:       lipgloss.Color("#ECEFF4"),
		Secondary:     lipgloss.Color("#E5E9F0"),
		Accent:        lipgloss.Color("#88C0D0"),
		Muted:         lipgloss.Color("#4C566A"),
		Good:          lipgloss.Color("#A3BE8C"),
		Warn:          lipgloss.Color("#EBCB8B"),
		Error:         lipgloss.Color("#BF616A"),
		GradientStart: lipgloss.Color("#81A1C1"),
		GradientEnd:   lipgloss.Color("#88C0D0"),
	},
	"tokyo-night": {
		Name:          "tokyo-night",
		Description:   "Dark cyberpunk blue with energetic accents",
		Primary:       lipgloss.Color("#C0CAF5"),
		Secondary:     lipgloss.Color("#A9B1D6"),
		Accent:        lipgloss.Color("#7AA2F7"),
		Muted:         lipgloss.Color("#565F89"),
		Good:          lipgloss.Color("#9ECE6A"),
		Warn:          lipgloss.Color("#E0AF68"),
		Error:         lipgloss.Color("#F7768E"),
		GradientStart: lipgloss.Color("#7AA2F7"),
		GradientEnd:   lipgloss.Color("#BB9AF7"),
	},
	"solarized": {
		Name:          "solarized",
		Description:   "Precision-engineered solarized palette",
		Primary:       lipgloss.Color("#EEE8D5"),
		Secondary:     lipgloss.Color("#93A1A1"),
		Accent:        lipgloss.Color("#268BD2"),
		Muted:         lipgloss.Color("#586E75"),
		Good:          lipgloss.Color("#859900"),
		Warn:          lipgloss.Color("#B58900"),
		Error:         lipgloss.Color("#DC322F"),
		GradientStart: lipgloss.Color("#268BD2"),
		GradientEnd:   lipgloss.Color("#2AA198"),
	},
}

func (t Theme) Gradient() (string, string) {
	start := string(t.GradientStart)
	if start == "" {
		start = string(t.Accent)
	}
	end := string(t.GradientEnd)
	if end == "" {
		end = string(t.Secondary)
	}
	if start == "" {
		start = "#5A56E0"
	}
	if end == "" {
		end = "#EE6FF8"
	}
	return start, end
}

func Get(name string) (Theme, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if t, ok := registry[name]; ok {
		return t, nil
	}
	return Theme{}, fmt.Errorf("unknown theme '%s'. Available themes: %s", name, strings.Join(Names(), ", "))
}

func Default() Theme {
	return registry["default"]
}

func Names() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func List() []Theme {
	names := Names()
	list := make([]Theme, len(names))
	for i, name := range names {
		list[i] = registry[name]
	}
	return list
}

func Preview(name string) (string, error) {
	t, err := Get(name)
	if err != nil {
		return "", err
	}

	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	primaryStyle := lipgloss.NewStyle().Foreground(t.Primary)
	secondaryStyle := lipgloss.NewStyle().Foreground(t.Secondary)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	goodStyle := lipgloss.NewStyle().Foreground(t.Good).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(t.Warn).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(t.Error).Bold(true)

	chip := func(color lipgloss.Color, label string) string {
		return lipgloss.NewStyle().
			Background(color).
			Foreground(lipgloss.Color("#111827")).
			Padding(0, 1).
			Bold(true).
			Render(label)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n\n", titleStyle.Render(t.Name), mutedStyle.Render(t.Description))
	b.WriteString("Color Swatches:\n  ")
	b.WriteString(chip(t.Accent, "Accent") + " ")
	b.WriteString(chip(t.Primary, "Primary") + " ")
	b.WriteString(chip(t.Secondary, "Secondary") + " ")
	b.WriteString(chip(t.Good, "Good") + " ")
	b.WriteString(chip(t.Warn, "Warn") + " ")
	b.WriteString(chip(t.Error, "Error") + " ")
	b.WriteString(chip(t.Muted, "Muted") + "\n\n")

	b.WriteString("Sample UI Elements:\n")
	fmt.Fprintf(&b, "  Status Badges:  %s  %s  %s\n",
		goodStyle.Render("● RUNNING"),
		warnStyle.Render("⏸ PAUSED"),
		errorStyle.Render("✕ ABANDONED"))
	fmt.Fprintf(&b, "  Session Clock:  %s\n", primaryStyle.Bold(true).Render("03:00")+" "+mutedStyle.Render("(Round 1/3)"))
	fmt.Fprintf(&b, "  Progress Bar:   %s\n",
		lipgloss.NewStyle().Foreground(t.Accent).Render("██████████████████░░░░░░░░░░░░  60%"))
	fmt.Fprintf(&b, "  Instruction:    %s\n", secondaryStyle.Render("inhale fully, exhale without forcing"))
	fmt.Fprintf(&b, "  Hotkeys:        %s\n", mutedStyle.Render("[space] pause  [enter] next  [q] quit"))

	return b.String(), nil
}
