package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Primary lipgloss.Color
	Muted   lipgloss.Color
	Accent  lipgloss.Color
	Good    lipgloss.Color
	Warn    lipgloss.Color
}

func themeByName(name string) Theme {
	switch name {
	case "catppuccin-mocha":
		return Theme{Primary: "#CDD6F4", Muted: "#6C7086", Accent: "#CBA6F7", Good: "#A6E3A1", Warn: "#F9E2AF"}
	default:
		return Theme{Primary: "#E5E7EB", Muted: "#6B7280", Accent: "#60A5FA", Good: "#34D399", Warn: "#FBBF24"}
	}
}
