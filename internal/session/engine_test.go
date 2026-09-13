package session

import (
	"testing"
	"time"
)

func TestTimedPhasesAndManualRetention(t *testing.T) {
	e := New(Settings{Rounds: 1, Breathing: 3 * time.Second, Recovery: 2 * time.Second})

	e.Tick(3 * time.Second)
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected retention, got %s", e.Phase())
	}

	e.Tick(5 * time.Second)
	if e.PhaseClock() != 5*time.Second {
		t.Fatalf("expected 5s retention, got %s", e.PhaseClock())
	}

	e.Advance()
	if e.Phase() != PhaseRecovery {
		t.Fatalf("expected recovery, got %s", e.Phase())
	}

	e.Tick(2 * time.Second)
	if !e.Done() {
		t.Fatal("expected session to complete")
	}

	results := e.Results()
	if len(results) != 1 || results[0].Retention != 5*time.Second {
		t.Fatalf("unexpected result: %#v", results)
	}
}

func TestPauseStopsBothClocks(t *testing.T) {
	e := New(Settings{Rounds: 1, Breathing: 10 * time.Second, Recovery: 2 * time.Second})
	e.Tick(2 * time.Second)
	e.TogglePause()
	e.Tick(5 * time.Second)
	if e.PhaseElapsed() != 2*time.Second || e.SessionElapsed() != 2*time.Second {
		t.Fatal("paused engine advanced")
	}
}

func TestRoundWaitWhenAutoNextDisabled(t *testing.T) {
	e := New(Settings{Rounds: 2, Breathing: time.Second, Recovery: time.Second, AutoNextRound: false})
	e.Tick(time.Second)
	e.Tick(4 * time.Second)
	e.Advance()
	e.Tick(time.Second)
	if e.Phase() != PhaseRoundReady {
		t.Fatalf("expected round ready, got %s", e.Phase())
	}
	e.Advance()
	if e.Phase() != PhaseBreathing || e.Round() != 2 {
		t.Fatalf("expected round 2 breathing, got round=%d phase=%s", e.Round(), e.Phase())
	}
}

func TestAutoNextStartsNextRound(t *testing.T) {
	e := New(Settings{Rounds: 2, Breathing: time.Second, Recovery: time.Second, AutoNextRound: true})
	e.Tick(time.Second)
	e.Advance()
	e.Tick(time.Second)
	if e.Round() != 2 || e.Phase() != PhaseBreathing {
		t.Fatalf("expected auto next round, got round=%d phase=%s", e.Round(), e.Phase())
	}
}
