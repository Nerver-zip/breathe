package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/Nerver-zip/breathing-tui/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// RenderBarChart7Days renders a clean open 7-day vertical bar chart matching pomo.
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

	labelStyle := lipgloss.NewStyle().Foreground(th.Muted)
	barStyle := lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	activeDayStyle := lipgloss.NewStyle().Foreground(th.Good).Bold(true)
	axisStyle := lipgloss.NewStyle().Foreground(th.Muted)

	var lines []string

	// Lines 1 & 2: blank lines to vertically align with Heatmap's Month Header & Divider lines
	lines = append(lines, "")
	lines = append(lines, "")

	// Rows 4 to 1: Data rows
	for row := 4; row >= 1; row-- {
		threshold := int(float64(row) / 4.0 * float64(maxRounds))
		var b strings.Builder
		fmt.Fprintf(&b, "%2d │", threshold)
		for i, d := range days {
			if d.Rounds >= threshold {
				b.WriteString(barStyle.Render(" ██"))
			} else if d.Rounds > 0 && row == 1 {
				b.WriteString(barStyle.Render(" ▄▄"))
			} else {
				b.WriteString("   ")
			}
			if i < len(days)-1 {
				b.WriteString(" ")
			} else {
				b.WriteString(" ")
			}
		}
		lines = append(lines, b.String())
	}

	// Line 7: Axis line ( 0 └───┬───┬───┬───┬───┬───┬────)
	var axis strings.Builder
	axis.WriteString(" 0 └")
	for i := range days {
		axis.WriteString("───")
		if i < len(days)-1 {
			axis.WriteString("┬")
		} else {
			axis.WriteString("─")
		}
	}
	lines = append(lines, axisStyle.Render(axis.String()))

	// Line 8: Day labels (Mon Tue Wed...)
	var dayLabels strings.Builder
	dayLabels.WriteString("    ")
	for i, d := range days {
		dayLabels.WriteString(labelStyle.Render(fmt.Sprintf("%-3s", d.DayLabel)))
		if i < len(days)-1 {
			dayLabels.WriteString(" ")
		} else {
			dayLabels.WriteString(" ")
		}
	}
	lines = append(lines, dayLabels.String())

	// Line 9: Rounds count line
	var countLine strings.Builder
	countLine.WriteString("    ")
	for i, d := range days {
		if d.Rounds > 0 {
			countLine.WriteString(activeDayStyle.Render(fmt.Sprintf("%3d", d.Rounds)))
		} else {
			countLine.WriteString(labelStyle.Render("  0"))
		}
		if i < len(days)-1 {
			countLine.WriteString(" ")
		} else {
			countLine.WriteString(" ")
		}
	}
	lines = append(lines, countLine.String())

	// Line 10: blank line to align with Heatmap's Legend line
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// RenderHeatmap renders a calendar-grouped ~4-month activity heatmap matching pomo.
func RenderHeatmap(heatmapDays []storage.HeatmapDay, th theme.Theme) string {
	if len(heatmapDays) == 0 {
		return ""
	}

	roundsMap := make(map[string]int, len(heatmapDays))
	for _, d := range heatmapDays {
		roundsMap[d.Date] = d.Rounds
	}

	refDate := time.Now()
	if len(heatmapDays) > 0 {
		if t, err := time.ParseInLocation("2006-01-02", heatmapDays[len(heatmapDays)-1].Date, time.Local); err == nil {
			refDate = t
		}
	}
	refDate = time.Date(refDate.Year(), refDate.Month(), refDate.Day(), 0, 0, 0, 0, time.Local)

	// 4 calendar months: 3 months ago up to current month
	months := make([]time.Time, 4)
	for i := 0; i < 4; i++ {
		months[i] = time.Date(refDate.Year(), refDate.Month()-time.Month(3-i), 1, 0, 0, 0, 0, refDate.Location())
	}

	labelStyle := lipgloss.NewStyle().Foreground(th.Muted)
	cellZero := lipgloss.NewStyle().Foreground(lipgloss.Color("#35374C")).Render("■ ")
	cellL1 := lipgloss.NewStyle().Foreground(th.Accent).Render("■ ")
	cellL2 := lipgloss.NewStyle().Foreground(th.Secondary).Render("■ ")
	cellL3 := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Render("■ ")
	cellL4 := lipgloss.NewStyle().Foreground(lipgloss.Color("#EE6FF8")).Render("■ ")

	renderCell := func(rounds int) string {
		switch {
		case rounds == 0:
			return cellZero
		case rounds <= 2:
			return cellL1
		case rounds <= 4:
			return cellL2
		case rounds <= 6:
			return cellL3
		default:
			return cellL4
		}
	}

	type monthGrid struct {
		name   string
		cols   [][]string // [colIndex][rowIndex 0..6 (Sun..Sat)]
		width  int
		header string
	}

	var mGrids []monthGrid
	for mIdx, mStart := range months {
		nextM := time.Date(mStart.Year(), mStart.Month()+1, 1, 0, 0, 0, 0, mStart.Location())
		lastDayOfM := nextM.AddDate(0, 0, -1).Day()

		maxDay := lastDayOfM
		if mIdx == 3 { // current month: only render up to today
			if refDate.Day() < maxDay {
				maxDay = refDate.Day()
			}
		}

		startWd := int(mStart.Weekday()) // 0=Sun, 6=Sat
		var cols [][]string
		currentCol := make([]string, 7)
		for r := 0; r < 7; r++ {
			currentCol[r] = "  "
		}
		row := startWd

		for day := 1; day <= maxDay; day++ {
			dStr := fmt.Sprintf("%04d-%02d-%02d", mStart.Year(), mStart.Month(), day)
			currentCol[row] = renderCell(roundsMap[dStr])
			row++
			if row > 6 {
				cols = append(cols, currentCol)
				currentCol = make([]string, 7)
				for r := 0; r < 7; r++ {
					currentCol[r] = "  "
				}
				row = 0
			}
		}
		if row > 0 {
			cols = append(cols, currentCol)
		}

		w := len(cols) * 2
		mName := mStart.Format("Jan")
		pad := w - len(mName)
		if pad < 0 {
			pad = 0
		}
		leftPad := pad / 2
		rightPad := pad - leftPad
		hStr := strings.Repeat(" ", leftPad) + mName + strings.Repeat(" ", rightPad)

		mGrids = append(mGrids, monthGrid{
			name:   mName,
			cols:   cols,
			width:  w,
			header: hStr,
		})
	}

	prefixLen := 6 // "Sun │ " is 6 chars
	var lines []string

	// Line 1: Month labels with calendar icon
	var monthHeader strings.Builder
	monthHeader.WriteString("🗓️    ")
	for i, mg := range mGrids {
		if i > 0 {
			monthHeader.WriteString("  ")
		}
		monthHeader.WriteString(mg.header)
	}
	lines = append(lines, labelStyle.Render(monthHeader.String()))

	// Calculate total visible width
	totalWidth := prefixLen
	for i, mg := range mGrids {
		if i > 0 {
			totalWidth += 2
		}
		totalWidth += mg.width
	}

	// Line 2: Divider line
	lines = append(lines, labelStyle.Render(strings.Repeat("─", totalWidth)))

	// Lines 3 to 9: 7 Day rows (Sun to Sat)
	dayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for r := 0; r < 7; r++ {
		var rowBuf strings.Builder
		rowBuf.WriteString(labelStyle.Render(fmt.Sprintf("%s │ ", dayNames[r])))
		for i, mg := range mGrids {
			if i > 0 {
				rowBuf.WriteString("  ")
			}
			for _, col := range mg.cols {
				rowBuf.WriteString(col[r])
			}
		}
		lines = append(lines, rowBuf.String())
	}

	// Line 10: Legend (Less ■ ■ ■ ■ ■ More)
	legendBlockZero := lipgloss.NewStyle().Foreground(lipgloss.Color("#35374C")).Render("■")
	legendBlock1 := lipgloss.NewStyle().Foreground(th.Accent).Render("■")
	legendBlock2 := lipgloss.NewStyle().Foreground(th.Secondary).Render("■")
	legendBlock3 := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Render("■")
	legendBlock4 := lipgloss.NewStyle().Foreground(lipgloss.Color("#EE6FF8")).Render("■")

	legendContent := fmt.Sprintf("%s %s %s %s %s %s %s",
		labelStyle.Render("Less"),
		legendBlockZero,
		legendBlock1,
		legendBlock2,
		legendBlock3,
		legendBlock4,
		labelStyle.Render("More"),
	)
	legendVisibleWidth := 19 // "Less ■ ■ ■ ■ ■ More"
	padLegend := totalWidth - legendVisibleWidth
	if padLegend < 0 {
		padLegend = 0
	}
	lines = append(lines, strings.Repeat(" ", padLegend)+legendContent)

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
	content.WriteString(fmt.Sprintf("  %s           Add +30s (timed) or +1 breath (counted)\n", keyStyle.Render("a        ")))
	content.WriteString(fmt.Sprintf("  %s           Subtract -1 breath (counted mode)\n", keyStyle.Render("s        ")))
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
