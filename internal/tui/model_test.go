package tui

import (
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
