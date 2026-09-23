package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/session"
	"github.com/Nerver-zip/breathing-tui/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelViewRendersStandardAndCompact(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:    3,
		Breathing: 3 * time.Minute,
		Recovery:  30 * time.Second,
	})

	m := New(engine, "default")

	// Standard terminal 80x24
	m.width = 80
	m.height = 24
	viewStandard := m.View()
	if !strings.Contains(viewStandard, "BREATHING TUI") || !strings.Contains(viewStandard, "DEEP BREATHING") {
		t.Fatalf("expected standard view to contain title and phase, got:\n%s", viewStandard)
	}

	// Compact terminal 50x14
	m.width = 50
	m.height = 14
	viewCompact := m.View()
	if !strings.Contains(viewCompact, "BREATHING TUI") || !strings.Contains(viewCompact, "DEEP BREATHING") {
		t.Fatalf("expected compact view to contain title and phase, got:\n%s", viewCompact)
	}
}

func TestModelOverlays(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:    2,
		Breathing: 5 * time.Second,
		Recovery:  5 * time.Second,
	})
	m := New(engine, "default")
	m.width = 80
	m.height = 24

	// Test '?' toggles help overlay
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)
	if !m.showHelp {
		t.Fatal("expected showHelp to be true")
	}
	helpView := m.View()
	if !strings.Contains(helpView, "HELP & SAFETY GUIDE") {
		t.Fatalf("expected help view, got:\n%s", helpView)
	}

	// Escape closes help
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = updated.(Model)
	if m.showHelp {
		t.Fatal("expected showHelp to be false after escape")
	}

	// Test 'q' opens quit confirmation overlay while active
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if !m.confirmQuit {
		t.Fatal("expected confirmQuit to be true")
	}
	quitView := m.View()
	if !strings.Contains(quitView, "ABANDON SESSION?") {
		t.Fatalf("expected quit modal, got:\n%s", quitView)
	}

	// 'n' cancels quit confirmation
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(Model)
	if m.confirmQuit {
		t.Fatal("expected confirmQuit to be false after 'n'")
	}
}

func TestModelResetConfirmationInRetention(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:    1,
		Breathing: 2 * time.Second,
		Recovery:  2 * time.Second,
	})
	m := New(engine, "default")
	m.width = 80
	m.height = 24

	// Advance to retention
	engine.Tick(2 * time.Second) // finishes breathing
	engine.Tick(3 * time.Second) // 3s into retention

	// Press 'r'
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(Model)
	if !m.confirmReset {
		t.Fatal("expected confirmReset to be true for non-zero retention")
	}
	modal := m.View()
	if !strings.Contains(modal, "RESET RETENTION?") {
		t.Fatalf("expected reset modal, got:\n%s", modal)
	}

	// Press 'y' to confirm reset
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	if m.confirmReset {
		t.Fatal("expected confirmReset to be false after confirmation")
	}
	if engine.PhaseElapsed() != 0 {
		t.Fatalf("expected phase elapsed to reset to 0, got %s", engine.PhaseElapsed())
	}
}

func TestCompletionSummaryView(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:    2,
		Breathing: 1 * time.Second,
		Recovery:  1 * time.Second,
	})
	// Complete round 1
	engine.Tick(1 * time.Second)  // breathing
	engine.Tick(10 * time.Second) // retention
	engine.Advance()              // recovery
	engine.Tick(1 * time.Second)  // round 1 complete

	// Complete round 2
	engine.Advance()              // round 2 breathing
	engine.Tick(1 * time.Second)  // breathing
	engine.Tick(15 * time.Second) // retention
	engine.Advance()              // recovery
	engine.Tick(1 * time.Second)  // session complete!

	if !engine.Done() {
		t.Fatal("expected engine to be done")
	}

	m := New(engine, "default")
	m.width = 80
	m.height = 24
	view := m.View()

	if !strings.Contains(view, "SESSION COMPLETE") {
		t.Fatalf("expected SESSION COMPLETE in summary, got:\n%s", view)
	}
	if !strings.Contains(view, "Round 1:") || !strings.Contains(view, "Round 2:") {
		t.Fatalf("expected round breakdowns in summary, got:\n%s", view)
	}
	if !strings.Contains(view, "Average Retention:") {
		t.Fatalf("expected average retention in summary, got:\n%s", view)
	}
}

func TestStatsModelView(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer store.Close()

	sm := NewStatsModel(store, "default")
	sm.width = 80
	sm.height = 24

	// Initially loading
	if !strings.Contains(sm.View(), "Loading") {
		t.Fatalf("expected loading view, got:\n%s", sm.View())
	}

	// Trigger loaded message with empty store
	cmd := sm.Init()
	msg := cmd()
	updated, _ := sm.Update(msg)
	sm = updated.(StatsModel)

	emptyView := sm.View()
	if !strings.Contains(emptyView, "No breathing practice recorded yet") {
		t.Fatalf("expected zero-data view, got:\n%s", emptyView)
	}
}

func TestStatsModelViewWithData(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	now := time.Now()
	sessID, err := store.CreateSession(ctx, 3, now)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if err := store.SaveRound(ctx, sessID, session.RoundResult{
		Index:     1,
		Breathing: 3 * time.Minute,
		Retention: 45 * time.Second,
		Recovery:  15 * time.Second,
	}); err != nil {
		t.Fatalf("SaveRound: %v", err)
	}
	if err := store.EndSession(ctx, sessID, now.Add(4*time.Minute), 4*time.Minute, 4*time.Minute, "completed"); err != nil {
		t.Fatalf("EndSession: %v", err)
	}

	sm := NewStatsModel(store, "pomo")
	sm.width = 80
	sm.height = 24

	cmd := sm.Init()
	msg := cmd()
	updated, _ := sm.Update(msg)
	sm = updated.(StatsModel)

	view := sm.View()

	if !strings.Contains(view, "Breathing statistics") {
		t.Fatalf("expected 'Breathing statistics' header, got:\n%s", view)
	}
	if !strings.Contains(view, "streak") {
		t.Fatalf("expected streak widget, got:\n%s", view)
	}
	if !strings.Contains(view, "TODAY") || !strings.Contains(view, "RETENTION") || !strings.Contains(view, "ALL TIME") {
		t.Fatalf("expected summary columns TODAY, RETENTION, ALL TIME, got:\n%s", view)
	}
	if !strings.Contains(view, "Sun │") || !strings.Contains(view, "Mon │") {
		t.Fatalf("expected heatmap weekday rows, got:\n%s", view)
	}
	if !strings.Contains(view, "Less") || !strings.Contains(view, "More") {
		t.Fatalf("expected heatmap legend, got:\n%s", view)
	}
	if !strings.Contains(view, "0 └───") {
		t.Fatalf("expected bar chart axis, got:\n%s", view)
	}
	if !strings.Contains(view, "Completed rounds/day") {
		t.Fatalf("expected bar chart unit label, got:\n%s", view)
	}
}

func TestManualRecoveryAdvancePersistsCompletedRound(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "manual-recovery.db"))
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	startedAt := time.Now()
	sessionID, err := store.CreateSession(ctx, 1, startedAt)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	engine := session.New(session.Settings{
		Rounds:    1,
		Breathing: 10 * time.Second,
		Recovery:  10 * time.Second,
	})
	m := NewSessionModel(engine, store, sessionID, startedAt, nil, "default", "")

	engine.Tick(5 * time.Second)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // end breathing
	m = updated.(Model)
	engine.Tick(45 * time.Second)                         // retention
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // end retention
	m = updated.(Model)
	engine.Tick(3 * time.Second)                          // partial recovery
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // explicitly finish round
	m = updated.(Model)

	if !engine.Done() {
		t.Fatal("expected final round to complete")
	}
	report, err := store.GetStats(ctx, startedAt)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if report.TodaySessions != 1 || report.TodayRounds != 1 || report.TodayBestRetention != 45*time.Second {
		t.Fatalf("manual recovery completion was not persisted: today sessions=%d rounds=%d best retention=%s",
			report.TodaySessions, report.TodayRounds, report.TodayBestRetention)
	}
	if report.Recent7Days[6].Rounds != 1 {
		t.Fatalf("expected today's chart count to be 1 round, got %d", report.Recent7Days[6].Rounds)
	}
}

func TestHeaderAndProgressBar(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:    3,
		Breathing: 10 * time.Second,
		Recovery:  10 * time.Second,
	})
	m := New(engine, "pomo")
	m.width = 80
	m.height = 24

	// Elapse 5 seconds (50% progress in breathing)
	engine.Tick(5 * time.Second)

	view := m.View()

	// Verify header line 1 and line 2
	if !strings.Contains(view, "BREATHING TUI") {
		t.Fatalf("expected BREATHING TUI in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Round 1/3") {
		t.Fatalf("expected Round 1/3 in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Active Session 00:05") {
		t.Fatalf("expected Active Session 00:05 in view, got:\n%s", view)
	}

	// Verify progress bar has percentage e.g. 50%
	if !strings.Contains(view, "50%") {
		t.Fatalf("expected 50%% in progress bar, got:\n%s", view)
	}

	// Verify progress bar uses block runes and no bracket enclosures around the bar
	lines := strings.Split(view, "\n")
	foundProgressBar := false
	for _, l := range lines {
		if strings.Contains(l, "50%") {
			foundProgressBar = true
			if strings.Contains(l, "[██") || strings.Contains(l, "░]") {
				t.Fatalf("expected modern progress bar without brackets, got line: %q", l)
			}
			if !strings.Contains(l, "█") || !strings.Contains(l, "░") {
				t.Fatalf("expected bar runes █ and ░, got line: %q", l)
			}
		}
	}
	if !foundProgressBar {
		t.Fatalf("failed to locate progress bar line with 50%% in view:\n%s", view)
	}
}

func TestKeyAAddsBonusTimeInTimedMode(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:    2,
		Breathing: 30 * time.Second,
		Recovery:  10 * time.Second,
		Mode:      session.BreathingModeTimed,
	})
	m := New(engine, "default")
	m.width = 80
	m.height = 24

	// Verify initial hotkeys show [a] +30s
	view := m.View()
	if !strings.Contains(view, "[a] +30s") {
		t.Fatalf("expected hotkey [a] +30s in timed mode, got:\n%s", view)
	}

	// Press 'a' to add 30s
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)

	if engine.BonusBreathing() != 30*time.Second {
		t.Fatalf("expected bonus 30s, got %s", engine.BonusBreathing())
	}
	if engine.PhaseClock() != 60*time.Second {
		t.Fatalf("expected phase clock to be 60s, got %s", engine.PhaseClock())
	}
}

func TestKeyAIncrementsBreathsInCountedMode(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:        1,
		Breathing:     2 * time.Minute,
		Recovery:      10 * time.Second,
		Mode:          session.BreathingModeCounted,
		TargetBreaths: 3,
	})
	m := New(engine, "default")
	m.width = 80
	m.height = 24

	// Verify hotkeys show [a] +1 and [s] -1
	view := m.View()
	if !strings.Contains(view, "[a] +1  [s] -1") {
		t.Fatalf("expected hotkeys '[a] +1  [s] -1' in counted mode, got:\n%s", view)
	}

	// Press 'a' (first breath)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if engine.Breaths() != 1 {
		t.Fatalf("expected 1 breath, got %d", engine.Breaths())
	}

	// Immediate repeat (key held down / spam within debounce window) should be BLOCKED!
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if engine.Breaths() != 1 {
		t.Fatalf("expected continuous key-repeat to be blocked by debounce, got %d", engine.Breaths())
	}

	// Simulate legitimate deliberate next press by clearing debounce timestamp
	m.lastActionTime = time.Time{}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if engine.Breaths() != 2 {
		t.Fatalf("expected 2 breaths, got %d", engine.Breaths())
	}

	// Press 's' to subtract 1 (decrement test)
	m.lastActionTime = time.Time{}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)
	if engine.Breaths() != 1 {
		t.Fatalf("expected breath count to decrease to 1 with 's', got %d", engine.Breaths())
	}

	// Press 'a' back to 2, then to 3 (target reached)
	m.lastActionTime = time.Time{}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if engine.Breaths() != 2 {
		t.Fatalf("expected 2 breaths, got %d", engine.Breaths())
	}

	m.lastActionTime = time.Time{}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)
	if engine.Breaths() != 3 {
		t.Fatalf("expected 3 breaths, got %d", engine.Breaths())
	}
	if engine.Phase() != session.PhaseRetention {
		t.Fatalf("expected auto-transition to PhaseRetention upon reaching target, got %s", engine.Phase())
	}
}

func TestQuotesTypewriterAndAlternation(t *testing.T) {
	engine := session.New(session.Settings{
		Rounds:    2,
		Breathing: 2 * time.Minute,
		Recovery:  10 * time.Second,
	})
	m := New(engine, "pomo")
	m.width = 80
	m.height = 24

	// Case 1: No quotes configured -> Nothing displayed
	if m.renderQuote() != "" {
		t.Fatalf("expected empty quote render when no quotes configured, got %q", m.renderQuote())
	}
	view := m.View()
	if strings.Contains(view, "▍") {
		t.Fatalf("expected no typewriter cursor when no quotes configured")
	}

	// Case 2: Configured with 2 quotes and 30s interval
	quotes := []string{
		"Breathe peacefully.",
		"Stay centered.",
	}
	m.SetQuotes(quotes, 30*time.Second)

	// Before any tick, quoteElapsed is 0 -> 0 characters typed, but cursor is visible
	rendered := m.renderQuote()
	if !strings.Contains(rendered, "▍") {
		t.Fatalf("expected cursor on initial typing state, got %q", rendered)
	}

	// Tick 5 * 45ms = 225ms -> exactly 5 characters of "Breathe peacefully." -> "Breat"
	start := time.Now()
	m.lastTick = start
	updated, _ := m.Update(tickMsg(start.Add(225 * time.Millisecond)))
	m = updated.(Model)

	rendered = m.renderQuote()
	if !strings.Contains(rendered, "Breat") {
		t.Fatalf("expected 'Breat' after 225ms, got %q", rendered)
	}

	// Tick to 2 seconds (fully typed "Breathe peacefully.")
	updated, _ = m.Update(tickMsg(start.Add(2 * time.Second)))
	m = updated.(Model)
	rendered = m.renderQuote()
	if !strings.Contains(rendered, "Breathe peacefully.") {
		t.Fatalf("expected full phrase 'Breathe peacefully.', got %q", rendered)
	}
	if strings.Contains(rendered, "▍") {
		t.Fatalf("expected cursor to disappear once typing completes, got %q", rendered)
	}

	// Pause test: while paused, quoteElapsed does not advance
	m.engine.SetPaused(true)
	prePauseElapsed := m.quoteElapsed
	updated, _ = m.Update(tickMsg(start.Add(10 * time.Second)))
	m = updated.(Model)
	if m.quoteElapsed != prePauseElapsed {
		t.Fatalf("expected quoteElapsed to stay frozen while paused, got %v vs %v", m.quoteElapsed, prePauseElapsed)
	}
	m.engine.SetPaused(false)

	// Alternate after 30 seconds of active time -> switches to quote 2: "Stay centered."
	// We were paused from 2s to 10s (8s paused).
	// Ticking to 40s gives 30s of additional active time (total 32s active >= 30s).
	updated, _ = m.Update(tickMsg(start.Add(40 * time.Second)))
	m = updated.(Model)
	if m.quoteIndex != 1 {
		t.Fatalf("expected quoteIndex to switch to 1 after 30s active time, got %d (elapsed %v)", m.quoteIndex, m.quoteElapsed)
	}

	// Next tick (200ms into quote 2) -> types first characters
	updated, _ = m.Update(tickMsg(start.Add(40200 * time.Millisecond)))
	m = updated.(Model)
	rendered = m.renderQuote()
	if !strings.Contains(rendered, "S") {
		t.Fatalf("expected quote 2 to begin typing 'S', got %q", rendered)
	}
}
