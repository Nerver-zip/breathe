package tui

import "strings"

var glyphs = map[rune][]string{
	'0': {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'1': {"  █  ", " ██  ", "  █  ", "  █  ", " ███ "},
	'2': {" ███ ", "    █", " ███ ", "█    ", "█████"},
	'3': {"████ ", "    █", " ███ ", "    █", "████ "},
	'4': {"█  █ ", "█  █ ", "█████", "   █ ", "   █ "},
	'5': {"█████", "█    ", "████ ", "    █", "████ "},
	'6': {" ███ ", "█    ", "████ ", "█   █", " ███ "},
	'7': {"█████", "    █", "   █ ", "  █  ", "  █  "},
	'8': {" ███ ", "█   █", " ███ ", "█   █", " ███ "},
	'9': {" ███ ", "█   █", " ████", "    █", " ███ "},
	':': {"     ", "  █  ", "     ", "  █  ", "     "},
}

func bigClock(value string) string {
	rows := make([]strings.Builder, 5)
	for _, r := range value {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		for i := range rows {
			if rows[i].Len() > 0 {
				rows[i].WriteString(" ")
			}
			rows[i].WriteString(g[i])
		}
	}
	out := make([]string, 5)
	for i := range rows {
		out[i] = rows[i].String()
	}
	return strings.Join(out, "\n")
}
