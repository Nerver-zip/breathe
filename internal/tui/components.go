package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/Nerver-zip/breathing-tui/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// RenderBarChart7Days renders a responsive 7-day vertical bar chart.
func RenderBarChart7Days(days []storage.DayActivity, th theme.Theme, maxWidth int) string {
	if len(days) == 0 {
		return ""
	}

	maxRounds := 0
	for _, d := range days {
		if d.Rounds > maxRounds {
			maxRounds = d.Rounds
		}
	}
	if maxRounds < 4 {
		maxRounds = 4
	}

	chartHeight := 5
	var lines []string

	labelStyle := lipgloss.NewStyle().Foreground(th.Muted)
	barStyle := lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	activeDayStyle := lipgloss.NewStyle().Foreground(th.Good).Bold(true)
	axisStyle := lipgloss.NewStyle().Foreground(th.Muted)

	for row := chartHeight; row >= 1; row-- {
		threshold := int(float64(row) / float64(chartHeight) * float64(maxRounds))
		var b strings.Builder
		fmt.Fprintf(&b, "%2d ┤ ", threshold)
		for _, d := range days {
			barChar := "  "
			if d.Rounds >= threshold {
				barChar = "██"
			} else if d.Rounds > 0 && row == 1 {
				barChar = "▄▄"
			}
			if d.Rounds > 0 {
				b.WriteString(barStyle.Render(barChar) + "  ")
			} else {
				b.WriteString(axisStyle.Render("··") + "  ")
			}
		}
		lines = append(lines, b.String())
	}

	// Axis line
	var axis strings.Builder
	axis.WriteString("   └─")
	for range days {
		axis.WriteString("────")
	}
	lines = append(lines, axisStyle.Render(axis.String()))

	// Day labels (Mon, Tue, ...)
	var dayLabels strings.Builder
	dayLabels.WriteString("     ")
	for _, d := range days {
		dayLabels.WriteString(labelStyle.Render(fmt.Sprintf("%-3s ", d.DayLabel)))
	}
	lines = append(lines, dayLabels.String())

	// Rounds count line
	var countLine strings.Builder
	countLine.WriteString("     ")
	for _, d := range days {
		if d.Rounds > 0 {
			countLine.WriteString(activeDayStyle.Render(fmt.Sprintf("%-3d ", d.Rounds)))
		} else {
			countLine.WriteString(labelStyle.Render(" 0  "))
		}
	}
	lines = append(lines, countLine.String())

	return strings.Join(lines, "\n")
}

// RenderHeatmap renders a GitHub-style ~4-month (18-week) activity heatmap.
func RenderHeatmap(heatmapDays []storage.HeatmapDay, th theme.Theme) string {
	if len(heatmapDays) == 0 {
		return ""
	}

	// 18 weeks x 7 days
	weeks := 18
	grid := make([][]storage.HeatmapDay, 7)
	for i := range grid {
		grid[i] = make([]storage.HeatmapDay, weeks)
	}

	// Days are sequential ending today. Map them into week columns.
	// We want column 0 to be the oldest week, column 17 to be the current week.
	for idx, d := range heatmapDays {
		if idx >= weeks*7 {
			break
		}
		col := idx / 7
		row := idx % 7
		if col < weeks && row < 7 {
			grid[row][col] = d
		}
	}

	dayLabels := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	labelStyle := lipgloss.NewStyle().Foreground(th.Muted)

	mutedDot := lipgloss.NewStyle().Foreground(th.Muted).Render("·")
	lowBlock := lipgloss.NewStyle().Foreground(th.Good).Render("░")
	medBlock := lipgloss.NewStyle().Foreground(th.Good).Bold(true).Render("▒")
	highBlock := lipgloss.NewStyle().Foreground(th.Accent).Bold(true).Render("▓")
	maxBlock := lipgloss.NewStyle().Foreground(th.Primary).Bold(true).Render("█")

	var lines []string

	// Header row: month labels
	var monthHeader strings.Builder
	monthHeader.WriteString("    ")
	var lastMonth time.Month
	for col := 0; col < weeks; col++ {
		d := grid[0][col]
		if d.Date != "" {
			t, err := time.Parse("2006-01-02", d.Date)
			if err == nil && t.Month() != lastMonth {
				monthHeader.WriteString(labelStyle.Render(t.Format("Jan ")))
				lastMonth = t.Month()
				continue
			}
		}
		if monthHeader.Len() < (col*2 + 4) {
			monthHeader.WriteString(" ")
		}
	}
	lines = append(lines, monthHeader.String())

	for row := 0; row < 7; row++ {
		var b strings.Builder
		// Show labels for Mon, Wed, Fri
		if row == 1 || row == 3 || row == 5 {
			fmt.Fprintf(&b, "%s ", labelStyle.Render(dayLabels[row]))
		} else {
			b.WriteString("    ")
		}

		for col := 0; col < weeks; col++ {
			day := grid[row][col]
			var cell string
			switch {
			case day.Rounds == 0:
				cell = mutedDot
			case day.Rounds <= 2:
				cell = lowBlock
			case day.Rounds <= 4:
				cell = medBlock
			case day.Rounds <= 6:
				cell = highBlock
			default:
				cell = maxBlock
			}
			b.WriteString(cell + " ")
		}
		lines = append(lines, b.String())
	}

	// Legend
	var legend strings.Builder
	legend.WriteString("    Less ")
	legend.WriteString(mutedDot + " ")
	legend.WriteString(lowBlock + " ")
	legend.WriteString(medBlock + " ")
	legend.WriteString(highBlock + " ")
	legend.WriteString(maxBlock + " More")
	lines = append(lines, labelStyle.Render(legend.String()))

	return strings.Join(lines, "\n")
}

// RenderHelpModal renders the popup help modal.
func RenderHelpModal(th theme.Theme, width, height int) string {
	boxWidth := 68
	if width > 0 && width-4 < boxWidth {
		boxWidth = width - 4
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	titleStyle := lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	headStyle := lipgloss.NewStyle().Foreground(th.Primary).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(th.Secondary)
	mutedStyle := lipgloss.NewStyle().Foreground(th.Muted)
	warnStyle := lipgloss.NewStyle().Foreground(th.Warn).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(th.Good).Bold(true)

	var content strings.Builder
	content.WriteString(titleStyle.Render("BREATHING TUI — HELP & SAFETY GUIDE") + "\n\n")

	content.WriteString(headStyle.Render("Controls:") + "\n")
	content.WriteString(fmt.Sprintf("  %s   Pause / resume session clocks\n", keyStyle.Render("Space / p")))
	content.WriteString(fmt.Sprintf("  %s   Advance phase / confirm next round\n", keyStyle.Render("Enter / n")))
	content.WriteString(fmt.Sprintf("  %s           Add +30s (timed) or count breath (counted)\n", keyStyle.Render("a        ")))
	content.WriteString(fmt.Sprintf("  %s           Restart current phase\n", keyStyle.Render("r        ")))
	content.WriteString(fmt.Sprintf("  %s           Toggle this help overlay\n", keyStyle.Render("?        ")))
	content.WriteString(fmt.Sprintf("  %s           Quit session (confirms if active)\n\n", keyStyle.Render("q / Esc  ")))

	content.WriteString(headStyle.Render("Session Phases:") + "\n")
	content.WriteString(textStyle.Render("  1. Deep Breathing: Timed countdown or breath-counted. Inhale fully, exhale without force.\n"))
	content.WriteString(textStyle.Render("  2. Retention: Count-up. Open-ended hold after exhale. Press Enter to end.\n"))
	content.WriteString(textStyle.Render("  3. Recovery Hold: Countdown. Take one deep breath in and hold.\n\n"))

	content.WriteString(warnStyle.Render("Safety Warning:") + "\n")
	content.WriteString(mutedStyle.Render("  Breath-hold exercises may cause dizziness, tingling, or fainting.\n"))
	content.WriteString(mutedStyle.Render("  Always practice in a safe seated or lying position. NEVER practice in\n"))
	content.WriteString(mutedStyle.Render("  water, while driving, or operating machinery. Stop if you feel unwell.\n"))
	content.WriteString(mutedStyle.Render("  This software is a timer and does not provide medical advice.\n\n"))

	content.WriteString(mutedStyle.Render("Press [?] or [Esc] to return to the session."))

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(1, 2).
		Width(boxWidth)

	return modalStyle.Render(content.String())
}

// RenderConfirmModal renders a confirmation overlay box.
func RenderConfirmModal(title, message, prompt string, th theme.Theme) string {
	boxWidth := 54
	titleStyle := lipgloss.NewStyle().Foreground(th.Warn).Bold(true)
	msgStyle := lipgloss.NewStyle().Foreground(th.Primary)
	promptStyle := lipgloss.NewStyle().Foreground(th.Good).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(th.Muted)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(title),
		"",
		msgStyle.Render(message),
		"",
		promptStyle.Render(prompt),
		"",
		mutedStyle.Render("Press [y] to confirm, [n / Esc] to cancel"),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Warn).
		Padding(1, 3).
		Width(boxWidth).
		Render(content)
}
