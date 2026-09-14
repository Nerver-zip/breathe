package tui

import (
	"context"
	"fmt"
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
	goodStyle := lipgloss.NewStyle().Foreground(m.theme.Good).Bold(true)

	// Zero data state
	if m.report.TotalSessions == 0 && m.report.TotalRounds == 0 {
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Accent).
			Padding(2, 4).
			Render(lipgloss.JoinVertical(
				lipgloss.Center,
				titleStyle.Render("BREATHING TUI — STATISTICS"),
				"",
				primaryStyle.Render("No breathing practice recorded yet."),
				mutedStyle.Render("Start your first session with:"),
				"",
				goodStyle.Render("breath start"),
				"",
				mutedStyle.Render("[?] help  [q] quit"),
			))
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
		}
		return box
	}

	// Card builder helper
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Muted).
		Padding(0, 1)

	header := titleStyle.Render("BREATHING TUI — STATISTICS DASHBOARD")

	// Today Card
	todayContent := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s\n%s  %s",
		mutedStyle.Render("Sessions:"), primaryStyle.Bold(true).Render(fmt.Sprintf("%d", m.report.TodaySessions)),
		mutedStyle.Render("Rounds:  "), primaryStyle.Bold(true).Render(fmt.Sprintf("%d", m.report.TodayRounds)),
		mutedStyle.Render("Active:  "), primaryStyle.Bold(true).Render(storage.FormatDuration(m.report.TodayActive)),
		mutedStyle.Render("Best:    "), goodStyle.Render(storage.FormatDuration(m.report.TodayBestRetention)),
	)
	todayCard := cardStyle.Width(17).Render(lipgloss.JoinVertical(lipgloss.Left, primaryStyle.Bold(true).Render("TODAY"), todayContent))

	// Retention Card
	retContent := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s",
		mutedStyle.Render("Average:"), primaryStyle.Bold(true).Render(storage.FormatDuration(m.report.AverageRetention)),
		mutedStyle.Render("Best:   "), goodStyle.Render(storage.FormatDuration(m.report.BestRetention)),
		mutedStyle.Render("Latest: "), primaryStyle.Bold(true).Render(storage.FormatDuration(m.report.LatestRetention)),
	)
	retCard := cardStyle.Width(17).Render(lipgloss.JoinVertical(lipgloss.Left, primaryStyle.Bold(true).Render("RETENTION"), retContent))

	// Streak Card
	streakContent := fmt.Sprintf("%s  %s\n%s  %s",
		mutedStyle.Render("Current:"), goodStyle.Render(fmt.Sprintf("%d days", m.report.CurrentStreak)),
		mutedStyle.Render("Best:   "), primaryStyle.Bold(true).Render(fmt.Sprintf("%d days", m.report.BestStreak)),
	)
	streakCard := cardStyle.Width(17).Render(lipgloss.JoinVertical(lipgloss.Left, primaryStyle.Bold(true).Render("STREAK"), streakContent))

	// All Time Card
	allContent := fmt.Sprintf("%s  %s\n%s  %s\n%s  %s",
		mutedStyle.Render("Sessions:"), primaryStyle.Bold(true).Render(fmt.Sprintf("%d", m.report.TotalSessions)),
		mutedStyle.Render("Rounds:  "), primaryStyle.Bold(true).Render(fmt.Sprintf("%d", m.report.TotalRounds)),
		mutedStyle.Render("Active:  "), primaryStyle.Bold(true).Render(storage.FormatDuration(m.report.TotalActive)),
	)
	allCard := cardStyle.Width(17).Render(lipgloss.JoinVertical(lipgloss.Left, primaryStyle.Bold(true).Render("ALL TIME"), allContent))

	// Row 1: Summary Cards
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, todayCard, " ", retCard, " ", streakCard, " ", allCard)

	// Row 2: Charts
	chart7Day := RenderBarChart7Days(m.report.Recent7Days, m.theme, 32)
	chartCard := cardStyle.Width(35).Render(lipgloss.JoinVertical(lipgloss.Left, primaryStyle.Bold(true).Render("7-DAY ACTIVITY (ROUNDS)"), chart7Day))

	heatmap := RenderHeatmap(m.report.HeatmapDays, m.theme)
	heatCard := cardStyle.Width(41).Render(lipgloss.JoinVertical(lipgloss.Left, primaryStyle.Bold(true).Render("ACTIVITY HEATMAP (~4 MONTHS)"), heatmap))

	row2 := lipgloss.JoinHorizontal(lipgloss.Top, chartCard, " ", heatCard)

	footer := mutedStyle.Render("[?] help overlay  [q / esc] quit")

	body := lipgloss.JoinVertical(lipgloss.Center, header, "", row1, "", row2, "", footer)

	if m.width <= 0 || m.height <= 0 {
		return body
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}
