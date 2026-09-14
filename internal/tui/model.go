package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/notify"
	"github.com/Nerver-zip/breathing-tui/internal/session"
	"github.com/Nerver-zip/breathing-tui/internal/storage"
	"github.com/Nerver-zip/breathing-tui/internal/theme"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type Model struct {
	engine               *session.Engine
	store                *storage.Store
	sessionID            int64
	startedAt            time.Time
	notifier             notify.Notifier
	theme                theme.Theme
	font                 string
	progressBar          progress.Model
	width                int
	height               int
	lastTick             time.Time
	quitting             bool
	abandoned            bool
	openStats            bool
	showHelp             bool
	confirmQuit          bool
	confirmReset         bool
	wasPausedBeforeModal bool
	lastActionTime       time.Time
}

func newProgressBar(th theme.Theme) progress.Model {
	startColor, endColor := th.Gradient()
	prog := progress.New(
		progress.WithGradient(startColor, endColor),
	)
	prog.Full = '█'
	prog.Empty = '░'
	prog.EmptyColor = string(th.Muted)
	prog.PercentageStyle = lipgloss.NewStyle().Foreground(th.Secondary)
	prog.PercentFormat = " %3.0f%%"
	prog.Width = 50
	return prog
}

func New(engine *session.Engine, themeName string) Model {
	th := themeByName(themeName)
	return Model{
		engine:      engine,
		theme:       th,
		font:        DefaultFont,
		startedAt:   time.Now(),
		notifier:    notify.New(notify.Options{Desktop: false, Bell: false}),
		progressBar: newProgressBar(th),
	}
}

func NewSessionModel(engine *session.Engine, store *storage.Store, sessionID int64, startedAt time.Time, notif notify.Notifier, themeName string, font string) Model {
	if font == "" {
		font = DefaultFont
	}
	th := themeByName(themeName)
	return Model{
		engine:      engine,
		store:       store,
		sessionID:   sessionID,
		startedAt:   startedAt,
		notifier:    notif,
		theme:       th,
		font:        font,
		progressBar: newProgressBar(th),
	}
}

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m *Model) processEvents() {
	events := m.engine.PopEvents()
	for _, ev := range events {
		if ev.Type == session.EventRecoveryComplete {
			// Save completed round to storage immediately
			results := m.engine.Results()
			if len(results) > 0 && m.store != nil && m.sessionID > 0 {
				lastRes := results[len(results)-1]
				_ = m.store.SaveRound(context.Background(), m.sessionID, lastRes)
			}
		}

		if ev.Auto {
			switch ev.Type {
			case session.EventBreathingComplete:
				if m.notifier != nil {
					m.notifier.Notify("Breathing Complete", "Begin retention hold")
				}
			case session.EventRecoveryComplete:
				if m.notifier != nil && !m.engine.Done() {
					m.notifier.Notify("Recovery Complete", fmt.Sprintf("Round %d ready", ev.Round+1))
				}
			case session.EventSessionComplete:
				if m.notifier != nil {
					m.notifier.Notify("Session Complete", "All breathing rounds finished")
				}
			}
		}

		if m.engine.Done() && m.store != nil && m.sessionID > 0 {
			_ = m.store.EndSession(context.Background(), m.sessionID, time.Now(), m.engine.SessionElapsed(), time.Since(m.startedAt), "completed")
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		barWidth := 50
		if m.width > 0 && m.width-8 < barWidth {
			barWidth = m.width - 8
		}
		if barWidth < 20 {
			barWidth = 20
		}
		m.progressBar.Width = barWidth
		return m, nil

	case tickMsg:
		now := time.Time(msg)
		if m.lastTick.IsZero() {
			m.lastTick = now
		} else {
			delta := now.Sub(m.lastTick)
			m.lastTick = now
			m.engine.Tick(delta)
			m.processEvents()
		}
		return m, tick()

	case tea.KeyMsg:
		key := msg.String()

		// Overlay: Quit Confirmation
		if m.confirmQuit {
			switch key {
			case "y", "Y":
				m.abandoned = true
				m.quitting = true
				if m.store != nil && m.sessionID > 0 {
					_ = m.store.EndSession(context.Background(), m.sessionID, time.Now(), m.engine.SessionElapsed(), time.Since(m.startedAt), "abandoned")
				}
				return m, tea.Quit
			case "n", "N", "esc", "q":
				m.confirmQuit = false
				m.engine.SetPaused(m.wasPausedBeforeModal)
				return m, nil
			default:
				return m, nil
			}
		}

		// Overlay: Reset Confirmation
		if m.confirmReset {
			switch key {
			case "y", "Y":
				m.engine.ResetCurrentPhase()
				m.confirmReset = false
				m.engine.SetPaused(m.wasPausedBeforeModal)
				return m, nil
			case "n", "N", "esc", "r":
				m.confirmReset = false
				m.engine.SetPaused(m.wasPausedBeforeModal)
				return m, nil
			default:
				return m, nil
			}
		}

		// Overlay: Help
		if m.showHelp {
			if key == "?" || key == "esc" || key == "q" {
				m.showHelp = false
				m.engine.SetPaused(m.wasPausedBeforeModal)
			}
			return m, nil
		}

		// Session Done Keys
		if m.engine.Done() {
			switch key {
			case "q", "enter", "esc", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "s":
				m.openStats = true
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		// Active Session Keys
		switch key {
		case "ctrl+c", "q":
			m.wasPausedBeforeModal = m.engine.Paused()
			m.engine.SetPaused(true)
			m.confirmQuit = true
			return m, nil

		case "a", "A", "+":
			if m.engine.Phase() == session.PhaseBreathing {
				if time.Since(m.lastActionTime) < 300*time.Millisecond {
					return m, nil
				}
				m.lastActionTime = time.Now()
				if m.engine.Mode() == session.BreathingModeTimed {
					m.engine.AddBreathingTime(30 * time.Second)
				} else if m.engine.Mode() == session.BreathingModeCounted {
					m.engine.IncrementBreaths()
					m.processEvents()
				}
			}
			return m, nil

		case "s", "S", "-":
			if m.engine.Phase() == session.PhaseBreathing && m.engine.Mode() == session.BreathingModeCounted {
				if time.Since(m.lastActionTime) < 300*time.Millisecond {
					return m, nil
				}
				m.lastActionTime = time.Now()
				m.engine.DecrementBreaths()
			}
			return m, nil

		case " ", "p":
			m.engine.TogglePause()
			return m, nil

		case "enter", "n":
			m.engine.Advance()
			m.processEvents()
			return m, nil

		case "r":
			// If in retention with active count-up, ask for confirmation
			if m.engine.Phase() == session.PhaseRetention && m.engine.PhaseElapsed() > 0 {
				m.wasPausedBeforeModal = m.engine.Paused()
				m.engine.SetPaused(true)
				m.confirmReset = true
			} else {
				m.engine.ResetCurrentPhase()
			}
			return m, nil

		case "?":
			m.wasPausedBeforeModal = m.engine.Paused()
			m.engine.SetPaused(true)
			m.showHelp = true
			return m, nil
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Overlays take precedence
	if m.confirmQuit {
		modal := RenderConfirmModal(
			"ABANDON SESSION?",
			"Leave active session?\nCompleted rounds will be preserved.",
			"Abandon session?",
			m.theme,
		)
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		}
		return modal
	}

	if m.confirmReset {
		modal := RenderConfirmModal(
			"RESET RETENTION?",
			fmt.Sprintf("Restart phase and discard %s retention time?", formatClock(m.engine.PhaseElapsed())),
			"Discard retention?",
			m.theme,
		)
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		}
		return modal
	}

	if m.showHelp {
		modal := RenderHelpModal(m.theme, m.width, m.height)
		if m.width > 0 && m.height > 0 {
			return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
		}
		return modal
	}

	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	primary := lipgloss.NewStyle().Foreground(m.theme.Primary)
	secondary := lipgloss.NewStyle().Foreground(m.theme.Secondary)
	muted := lipgloss.NewStyle().Foreground(m.theme.Muted)
	good := lipgloss.NewStyle().Foreground(m.theme.Good)
	warn := lipgloss.NewStyle().Foreground(m.theme.Warn)

	// Completion Summary View
	if m.engine.Done() {
		return m.viewSummary()
	}

	phaseTitle, instruction := phaseCopy(m.engine.Phase())
	if m.engine.Phase() == session.PhaseBreathing && m.engine.Mode() == session.BreathingModeCounted {
		instruction = "inhale fully, exhale without forcing • press [a] after each breath"
	}

	clockBlock := ""
	f := m.font
	if f == "" {
		f = DefaultFont
	}

	if m.engine.Phase() == session.PhaseBreathing && m.engine.Mode() == session.BreathingModeCounted {
		breathsStr := fmt.Sprintf("%02d", m.engine.Breaths())
		if m.width >= 60 && m.height >= 18 {
			bigDigits := primary.Bold(true).Render(RenderClock(breathsStr, f))
			subText := muted.Render(fmt.Sprintf("breaths %d / %d   •   elapsed %s",
				m.engine.Breaths(), m.engine.TargetBreaths(), formatClock(m.engine.PhaseElapsed())))
			clockBlock = lipgloss.JoinVertical(lipgloss.Center, bigDigits, subText)
		} else {
			clockBlock = primary.Bold(true).Render(fmt.Sprintf("  🫁  BREATHS %d / %d  (%s)  ",
				m.engine.Breaths(), m.engine.TargetBreaths(), formatClock(m.engine.PhaseElapsed())))
		}
	} else {
		clock := formatClock(m.engine.PhaseClock())
		if m.width >= 60 && m.height >= 18 {
			clockBlock = primary.Bold(true).Render(RenderClock(clock, f))
		} else {
			clockBlock = primary.Bold(true).Render(fmt.Sprintf("  ⏱  %s  ", clock))
		}
	}

	status := "RUNNING"
	statusStyle := good
	if m.engine.Paused() {
		status = "PAUSED"
		statusStyle = warn
	} else if m.engine.Phase() == session.PhaseRoundReady {
		status = "ROUND COMPLETE"
		statusStyle = warn
	}

	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	leftTitle := titleStyle.Render("BREATHING TUI")
	roundText := muted.Render(fmt.Sprintf("Round %d/%d", m.engine.Round(), m.engine.TotalRounds()))
	sessionText := muted.Render(fmt.Sprintf("Active Session %s", formatClock(m.engine.SessionElapsed())))

	margin := 2
	innerW := w - (margin * 2)
	if innerW < 30 {
		innerW = 30
	}

	gap1 := innerW - lipgloss.Width(leftTitle) - lipgloss.Width(roundText)
	if gap1 < 1 {
		gap1 = 1
	}
	line1 := strings.Repeat(" ", margin) + leftTitle + strings.Repeat(" ", gap1) + roundText

	gap2 := innerW - lipgloss.Width(sessionText)
	if gap2 < 0 {
		gap2 = 0
	}
	line2 := strings.Repeat(" ", margin) + strings.Repeat(" ", gap2) + sessionText

	header := line1 + "\n" + line2

	progressBarView := m.renderProgressBar()

	hotkeysText := "[space/p] pause  [enter/n] next  [r] reset  [?] help  [q] quit"
	if m.engine.Phase() == session.PhaseBreathing {
		if m.engine.Mode() == session.BreathingModeCounted {
			hotkeysText = "[a] +1  [s] -1  [space/p] pause  [enter/n] next  [r] reset  [?] help  [q] quit"
		} else {
			hotkeysText = "[a] +30s  [space/p] pause  [enter/n] next  [r] reset  [?] help  [q] quit"
		}
	}
	hotkeys := muted.Render(hotkeysText)

	centerContent := lipgloss.JoinVertical(
		lipgloss.Center,
		statusStyle.Bold(true).Render("● "+status),
		primary.Bold(true).Render(phaseTitle),
		secondary.Render(instruction),
		"",
		clockBlock,
		"",
		progressBarView,
		"",
		hotkeys,
	)

	if m.width <= 0 || m.height <= 0 {
		return lipgloss.JoinVertical(lipgloss.Left, header, "", centerContent)
	}

	availH := h - 3
	if availH < lipgloss.Height(centerContent) {
		return lipgloss.JoinVertical(lipgloss.Left, header, "", centerContent)
	}

	placedCenter := lipgloss.Place(w, availH, lipgloss.Center, lipgloss.Center, centerContent)
	return header + "\n\n" + placedCenter
}

func (m Model) renderProgressBar() string {
	switch m.engine.Phase() {
	case session.PhaseBreathing, session.PhaseRecovery:
		barWidth := 50
		if m.width > 0 && m.width-8 < barWidth {
			barWidth = m.width - 8
		}
		if barWidth < 20 {
			barWidth = 20
		}
		m.progressBar.Width = barWidth
		return m.progressBar.ViewAs(m.engine.Progress())
	case session.PhaseRetention:
		muted := lipgloss.NewStyle().Foreground(m.theme.Muted)
		return muted.Render("retention is open-ended — press [enter] when you need to breathe")
	case session.PhaseRoundReady:
		good := lipgloss.NewStyle().Foreground(m.theme.Good).Bold(true)
		return good.Render("press [enter] to begin next round")
	default:
		return ""
	}
}

func (m Model) viewSummary() string {
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	primary := lipgloss.NewStyle().Foreground(m.theme.Primary)
	secondary := lipgloss.NewStyle().Foreground(m.theme.Secondary)
	good := lipgloss.NewStyle().Foreground(m.theme.Good).Bold(true)
	muted := lipgloss.NewStyle().Foreground(m.theme.Muted)

	results := m.engine.Results()
	var totalRetention time.Duration
	var bestRetention time.Duration

	var table strings.Builder
	for i, r := range results {
		totalRetention += r.Retention
		if r.Retention > bestRetention {
			bestRetention = r.Retention
		}
		deltaStr := ""
		if i > 0 {
			diff := r.Retention - results[i-1].Retention
			if diff > 0 {
				deltaStr = good.Render(fmt.Sprintf(" (+%s)", formatClock(diff)))
			} else if diff < 0 {
				deltaStr = muted.Render(fmt.Sprintf(" (-%s)", formatClock(-diff)))
			} else {
				deltaStr = muted.Render(" (=)")
			}
		}
		breathsStr := ""
		if r.Breaths > 0 {
			breathsStr = muted.Render(fmt.Sprintf(" (%d breaths)", r.Breaths))
		}
		fmt.Fprintf(&table, "  Round %d:   %s%s%s\n", r.Index, primary.Bold(true).Render(formatClock(r.Retention)), deltaStr, breathsStr)
	}

	var avgRetention time.Duration
	if len(results) > 0 {
		avgRetention = totalRetention / time.Duration(len(results))
	}

	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Good).
		Padding(1, 3).
		Render(lipgloss.JoinVertical(
			lipgloss.Center,
			good.Render("✔ SESSION COMPLETE"),
			"",
			secondary.Render(fmt.Sprintf("Completed %d of %d planned rounds", len(results), m.engine.TotalRounds())),
			secondary.Render(fmt.Sprintf("Total Active Time: %s", formatClock(m.engine.SessionElapsed()))),
			"",
			table.String(),
			muted.Render(fmt.Sprintf("Average Retention: %s   •   Best: %s", formatClock(avgRetention), formatClock(bestRetention))),
			"",
			muted.Render("[enter / q] exit   •   [s] view full statistics"),
		))

	body := lipgloss.JoinVertical(lipgloss.Center, titleStyle.Render("BREATHING TUI"), "", summaryBox)

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
		return "DONE", "session saved to history"
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

func (m Model) Engine() *session.Engine { return m.engine }
func (m Model) ShouldOpenStats() bool   { return m.openStats }
func (m Model) IsAbandoned() bool       { return m.abandoned }
