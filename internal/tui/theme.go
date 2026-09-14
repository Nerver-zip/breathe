package tui

import "github.com/Nerver-zip/breathing-tui/internal/theme"

type Theme = theme.Theme

func themeByName(name string) Theme {
	t, err := theme.Get(name)
	if err != nil {
		return theme.Default()
	}
	return t
}
