package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/Nerver-zip/breathing-tui/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type statsLoadedMsg struct {
	report storage.StatsReport
	err    error
}

type StatsModel struct {
	store    *storage.Store
	report   storage.StatsReport
	theme    theme.Theme
	width    int
	height   int
	loading  bool
	err      error
	showHelp bool
	quitting bool
}

func NewStatsModel(store *storage.Store, themeName string) StatsModel {
	th := themeByName(themeName)
	return StatsModel{
		store:   store,
		theme:   th,
		loading: true,
	}
}

func (m StatsModel) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		rep, err := m.store.GetStats(ctx, time.Now())
		return statsLoadedMsg{report: rep, err: err}
	}
}

func (m StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case statsLoadedMsg:
		m.loading = false
		m.report = msg.report
		m.err = msg.err
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q", "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
		}
	}
	return m, nil
}

func (m StatsModel) View() string {
	if m.quitting {
		return ""
	}
	if m.showHelp {
		modal := RenderHelpModal(m.theme, m.width, m.height)
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		}
		return modal
	}

	if m.loading {
		msg := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Loading breathing statistics...")
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
		}
		return msg
	}

	if m.err != nil {
		msg := lipgloss.NewStyle().Foreground(m.theme.Error).Render(fmt.Sprintf("Failed to load statistics: %v", m.err))
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
		}
		return msg
	}

	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
	primaryStyle := lipgloss.NewStyle().Foreground(m.theme.Primary)
	secondaryStyle := lipgloss.NewStyle().Foreground(m.theme.Secondary)
	goodStyle := lipgloss.NewStyle().Foreground(m.theme.Good).Bold(true)

	// Zero data state
	if m.report.TotalSessions == 0 && m.report.TotalRounds == 0 {
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Accent).
			Padding(2, 4).
			Render(lipgloss.JoinVertical(
				lipgloss.Center,
				titleStyle.Render("Breathing statistics"),
				"",
				primaryStyle.Render("No breathing practice recorded yet."),
				mutedStyle.Render("Start your first session with:"),
				"",
				goodStyle.Render("breath start"),
				"",
				mutedStyle.Render("[?] help  •  [q] quit"),
			))
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
		}
		return box
	}

	header := titleStyle.Render("Breathing statistics")

	// Top Section: Progress bar & Streak
	barWidth := 36
	activeMS := m.report.TotalActive.Milliseconds()
	retMS := m.report.TotalRetention.Milliseconds()
	if retMS == 0 && m.report.TotalRounds > 0 {
		retMS = m.report.AverageRetention.Milliseconds() * int64(m.report.TotalRounds)
	}

	var activePct, retPct int
	if activeMS > 0 {
		retPct = int(float64(retMS) / float64(activeMS) * 100)
		if retPct > 100 {
			retPct = 100
		}
		activePct = 100 - retPct
		if retPct == 0 && retMS > 0 {
			retPct = 1
			activePct = 99
		}
	} else {
		activePct = 100
		retPct = 0
	}

	activeLabel := formatCompactDuration(m.report.TotalActive)
	retLabel := formatCompactDuration(time.Duration(retMS) * time.Millisecond)

	padTop := barWidth - len(activeLabel) - len(retLabel)
	if padTop < 1 {
		padTop = 1
	}
	topLabels := primaryStyle.Bold(true).Render(activeLabel) + strings.Repeat(" ", padTop) + secondaryStyle.Bold(true).Render(retLabel)

	filledLen := int(float64(activePct) / 100.0 * float64(barWidth))
	if filledLen > barWidth {
		filledLen = barWidth
	}
	shadedLen := barWidth - filledLen

	filledBar := lipgloss.NewStyle().Foreground(m.theme.Accent).Render(strings.Repeat("█", filledLen))
	shadedBar := lipgloss.NewStyle().Foreground(lipgloss.Color("#35374C")).Render(strings.Repeat("░", shadedLen))
	barWidget := filledBar + shadedBar

	activePctStr := fmt.Sprintf("%d%%", activePct)
	retPctStr := fmt.Sprintf("%d%%", retPct)
	padBottom := barWidth - len(activePctStr) - len(retPctStr)
	if padBottom < 1 {
		padBottom = 1
	}
	bottomLabels := mutedStyle.Render(activePctStr) + strings.Repeat(" ", padBottom) + mutedStyle.Render(retPctStr)

	streakText := fmt.Sprintf("⚡ streak %dd • best %dd", m.report.CurrentStreak, m.report.BestStreak)
	streakWidget := lipgloss.NewStyle().Foreground(m.theme.Primary).Render(streakText)

	topSection := lipgloss.JoinVertical(
		lipgloss.Center,
		topLabels,
		barWidget,
		bottomLabels,
		"",
		streakWidget,
	)

	// Middle Section: Open 3-column summary
	colHeaderStyle := primaryStyle.Bold(true)
	renderRow := func(label, val string, valStyle lipgloss.Style) string {
		return mutedStyle.Render(fmt.Sprintf("%-11s", label)) + " " + valStyle.Render(fmt.Sprintf("%8s", val))
	}

	col1 := lipgloss.JoinVertical(
		lipgloss.Left,
		colHeaderStyle.Render("       TODAY        "),
		renderRow("Rounds", fmt.Sprintf("%d", m.report.TodayRounds), primaryStyle.Bold(true)),
		renderRow("Active", storage.FormatDuration(m.report.TodayActive), primaryStyle.Bold(true)),
		renderRow("Sessions", fmt.Sprintf("%d", m.report.TodaySessions), primaryStyle.Bold(true)),
		renderRow("Best Hold", storage.FormatDuration(m.report.TodayBestRetention), goodStyle),
	)

	col2 := lipgloss.JoinVertical(
		lipgloss.Left,
		colHeaderStyle.Render("     RETENTION      "),
		renderRow("Average", storage.FormatDuration(m.report.AverageRetention), primaryStyle.Bold(true)),
		renderRow("Best", storage.FormatDuration(m.report.BestRetention), goodStyle),
		renderRow("Latest", storage.FormatDuration(m.report.LatestRetention), primaryStyle.Bold(true)),
		renderRow("", "", mutedStyle),
	)

	col3 := lipgloss.JoinVertical(
		lipgloss.Left,
		colHeaderStyle.Render("      ALL TIME      "),
		renderRow("Rounds", fmt.Sprintf("%d", m.report.TotalRounds), primaryStyle.Bold(true)),
		renderRow("Active", storage.FormatDuration(m.report.TotalActive), primaryStyle.Bold(true)),
		renderRow("Sessions", fmt.Sprintf("%d", m.report.TotalSessions), primaryStyle.Bold(true)),
		renderRow("", "", mutedStyle),
	)

	summaryRow := lipgloss.JoinHorizontal(lipgloss.Top, col1, "      ", col2, "      ", col3)

	// Bottom Section: Charts
	chart7Day := RenderBarChart7Days(m.report.Recent7Days, m.theme, 32)
	heatmap := RenderHeatmap(m.report.HeatmapDays, m.theme)
	chartsRow := lipgloss.JoinHorizontal(lipgloss.Top, chart7Day, "    ", heatmap)

	footer := mutedStyle.Render("[?] help  •  [q / esc] quit")

	body := lipgloss.JoinVertical(
		lipgloss.Center,
		header,
		"",
		topSection,
		"",
		summaryRow,
		"",
		chartsRow,
		"",
		footer,
	)

	if m.width <= 0 || m.height <= 0 {
		return body
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

func formatCompactDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	s := int(d.Round(time.Second).Seconds())
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, sec)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, sec)
	}
	return fmt.Sprintf("%ds", sec)
}
