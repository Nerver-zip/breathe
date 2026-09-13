package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/session"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type Model struct {
	engine   *session.Engine
	theme    Theme
	width    int
	height   int
	lastTick time.Time
	quitting bool
}

func New(engine *session.Engine, themeName string) Model {
	return Model{engine: engine, theme: themeByName(themeName)}
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Init() tea.Cmd { return tick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		now := time.Time(msg)
		if m.lastTick.IsZero() {
			m.lastTick = now
		} else {
			m.engine.Tick(now.Sub(m.lastTick))
			m.lastTick = now
		}
		return m, tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case " ", "p":
			m.engine.TogglePause()
		case "enter", "n":
			m.engine.Advance()
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	primary := lipgloss.NewStyle().Foreground(m.theme.Primary)
	muted := lipgloss.NewStyle().Foreground(m.theme.Muted)
	good := lipgloss.NewStyle().Foreground(m.theme.Good)
	warn := lipgloss.NewStyle().Foreground(m.theme.Warn)

	phaseTitle, instruction := phaseCopy(m.engine.Phase())
	clock := formatClock(m.engine.PhaseClock())
	clockBlock := primary.Bold(true).Render(bigClock(clock))

	status := "RUNNING"
	statusStyle := good
	if m.engine.Paused() {
		status = "PAUSED"
		statusStyle = warn
	}
	if m.engine.Phase() == session.PhaseRoundReady {
		status = "ROUND COMPLETE"
		statusStyle = warn
	}
	if m.engine.Done() {
		status = "SESSION COMPLETE"
		statusStyle = good
	}

	progress := ""
	if m.engine.Phase() == session.PhaseBreathing || m.engine.Phase() == session.PhaseRecovery {
		progress = renderProgress(m.engine.Progress(), 42)
	} else if m.engine.Phase() == session.PhaseRetention {
		progress = muted.Render("retention is open-ended — advance when you need to breathe")
	}

	body := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("BREATHING TUI"),
		muted.Render(fmt.Sprintf("Round %d/%d  •  Session %s", m.engine.Round(), m.engine.TotalRounds(), formatClock(m.engine.SessionElapsed()))),
		"",
		statusStyle.Bold(true).Render(status),
		primary.Bold(true).Render(phaseTitle),
		muted.Render(instruction),
		"",
		clockBlock,
		"",
		progress,
		"",
		muted.Render("[space/p] pause  [enter/n] next  [q] quit"),
	)

	if m.engine.Done() {
		body = lipgloss.JoinVertical(lipgloss.Center, body, "", primary.Render(summaryText(m.engine.Results())))
	}

	if m.width <= 0 || m.height <= 0 {
		return body
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, body)
}

func phaseCopy(p session.Phase) (string, string) {
	switch p {
	case session.PhaseBreathing:
		return "DEEP BREATHING", "inhale fully, exhale without forcing"
	case session.PhaseRetention:
		return "RETENTION", "after the exhale, hold comfortably"
	case session.PhaseRecovery:
		return "RECOVERY HOLD", "inhale once, then hold"
	case session.PhaseRoundReady:
		return "READY FOR NEXT ROUND", "press Enter to begin the next round"
	case session.PhaseComplete:
		return "DONE", "session saved when you leave"
	default:
		return string(p), ""
	}
}

func renderProgress(progress float64, width int) string {
	if width < 5 {
		width = 5
	}
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func formatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Round(time.Second).Seconds())
	minutes := total / 60
	seconds := total % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func summaryText(results []session.RoundResult) string {
	if len(results) == 0 {
		return "No completed rounds."
	}
	var b strings.Builder
	b.WriteString("Retention: ")
	for i, r := range results {
		if i > 0 {
			b.WriteString("  •  ")
		}
		fmt.Fprintf(&b, "R%d %s", r.Index, formatClock(r.Retention))
	}
	return b.String()
}

func (m Model) Engine() *session.Engine { return m.engine }
