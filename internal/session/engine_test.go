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

func TestOversizeTickBreathingToRetention(t *testing.T) {
	e := New(Settings{Rounds: 1, Breathing: 2 * time.Second, Recovery: 2 * time.Second})
	// Tick 5 seconds when breathing is 2s: 2s should be consumed by breathing, 3s by retention
	e.Tick(5 * time.Second)

	if e.Phase() != PhaseRetention {
		t.Fatalf("expected PhaseRetention, got %s", e.Phase())
	}
	if e.PhaseClock() != 3*time.Second {
		t.Fatalf("expected retention clock 3s, got %s", e.PhaseClock())
	}
	if e.SessionElapsed() != 5*time.Second {
		t.Fatalf("expected session elapsed 5s, got %s", e.SessionElapsed())
	}

	events := e.PopEvents()
	if len(events) != 1 || events[0].Type != EventBreathingComplete || !events[0].Auto {
		t.Fatalf("expected auto EventBreathingComplete, got %#v", events)
	}
}

func TestOversizeTickRecoveryAutoNext(t *testing.T) {
	e := New(Settings{Rounds: 2, Breathing: 2 * time.Second, Recovery: 2 * time.Second, AutoNextRound: true})
	e.Tick(2 * time.Second) // finishes breathing round 1
	e.PopEvents()
	e.Advance() // advances to recovery round 1
	// 5s tick: 2s finishes recovery round 1, then transitions to round 2 breathing (2s), then 1s retention!
	e.Tick(5 * time.Second)

	if e.Round() != 2 {
		t.Fatalf("expected round 2, got %d", e.Round())
	}
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected PhaseRetention in round 2, got %s", e.Phase())
	}
	if e.PhaseClock() != 1*time.Second {
		t.Fatalf("expected round 2 retention clock 1s, got %s", e.PhaseClock())
	}
	if len(e.Results()) != 1 {
		t.Fatalf("expected 1 completed round result, got %d", len(e.Results()))
	}
}

func TestOversizeTickRecoveryFinalRound(t *testing.T) {
	e := New(Settings{Rounds: 1, Breathing: 2 * time.Second, Recovery: 2 * time.Second})
	e.Tick(2 * time.Second) // finishes breathing
	e.Advance()             // recovery
	e.PopEvents()

	// 10s tick: 2s completes recovery, remaining 8s dropped because session completes
	e.Tick(10 * time.Second)
	if !e.Done() {
		t.Fatalf("expected session to be done")
	}
	if e.SessionElapsed() != 4*time.Second {
		t.Fatalf("expected session elapsed 4s, got %s", e.SessionElapsed())
	}
	events := e.PopEvents()
	var hasSessionComplete bool
	for _, ev := range events {
		if ev.Type == EventSessionComplete {
			hasSessionComplete = true
		}
	}
	if !hasSessionComplete {
		t.Fatalf("expected EventSessionComplete, got %#v", events)
	}
}

func TestResetCurrentPhase(t *testing.T) {
	t.Run("reset breathing", func(t *testing.T) {
		e := New(Settings{Rounds: 2, Breathing: 10 * time.Second, Recovery: 5 * time.Second})
		e.Tick(4 * time.Second)
		if e.SessionElapsed() != 4*time.Second || e.PhaseElapsed() != 4*time.Second {
			t.Fatalf("unexpected elapsed before reset")
		}
		if !e.ResetCurrentPhase() {
			t.Fatal("expected ResetCurrentPhase to succeed")
		}
		if e.SessionElapsed() != 0 || e.PhaseElapsed() != 0 {
			t.Fatalf("expected 0 elapsed after reset, got session=%s phase=%s", e.SessionElapsed(), e.PhaseElapsed())
		}
	})

	t.Run("reset retention", func(t *testing.T) {
		e := New(Settings{Rounds: 2, Breathing: 3 * time.Second, Recovery: 5 * time.Second})
		e.Tick(3 * time.Second) // enters retention
		e.Tick(7 * time.Second) // 7s retention
		if e.SessionElapsed() != 10*time.Second || e.PhaseElapsed() != 7*time.Second {
			t.Fatalf("unexpected elapsed before reset")
		}
		if !e.ResetCurrentPhase() {
			t.Fatal("expected ResetCurrentPhase to succeed")
		}
		if e.SessionElapsed() != 3*time.Second || e.PhaseElapsed() != 0 {
			t.Fatalf("expected session=3s phase=0, got session=%s phase=%s", e.SessionElapsed(), e.PhaseElapsed())
		}
	})

	t.Run("reset recovery", func(t *testing.T) {
		e := New(Settings{Rounds: 1, Breathing: 3 * time.Second, Recovery: 5 * time.Second})
		e.Tick(3 * time.Second) // enters retention
		e.Tick(5 * time.Second) // 5s retention
		e.Advance()             // recovery
		e.Tick(2 * time.Second) // 2s recovery
		if e.SessionElapsed() != 10*time.Second {
			t.Fatalf("unexpected elapsed before reset: %s", e.SessionElapsed())
		}
		if !e.ResetCurrentPhase() {
			t.Fatal("expected ResetCurrentPhase to succeed")
		}
		if e.SessionElapsed() != 8*time.Second || e.PhaseElapsed() != 0 {
			t.Fatalf("expected session=8s phase=0, got session=%s phase=%s", e.SessionElapsed(), e.PhaseElapsed())
		}
	})
}

func TestDefaultSettingsClamping(t *testing.T) {
	e := New(Settings{Rounds: 0, Breathing: 0, Recovery: 0})
	if e.TotalRounds() != 1 {
		t.Fatalf("expected clamped rounds 1, got %d", e.TotalRounds())
	}
	if e.BreathingDuration() != 3*time.Minute {
		t.Fatalf("expected default breathing 3m, got %s", e.BreathingDuration())
	}
	if e.RecoveryDuration() != 30*time.Second {
		t.Fatalf("expected default recovery 30s, got %s", e.RecoveryDuration())
	}
}

func TestEarlyAdvanceDuringPhases(t *testing.T) {
	e := New(Settings{Rounds: 2, Breathing: 10 * time.Second, Recovery: 10 * time.Second})
	// Advance early during breathing at 2s
	e.Tick(2 * time.Second)
	e.Advance()
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected retention, got %s", e.Phase())
	}
	// Hold retention for 4s
	e.Tick(4 * time.Second)
	e.Advance()
	if e.Phase() != PhaseRecovery {
		t.Fatalf("expected recovery, got %s", e.Phase())
	}
	// Advance early during recovery at 3s
	e.Tick(3 * time.Second)
	e.Advance()
	results := e.Results()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Breathing != 2*time.Second || r.Retention != 4*time.Second || r.Recovery != 3*time.Second {
		t.Fatalf("unexpected round 1 result: %#v", r)
	}
}

func TestAddBreathingTimeTimedMode(t *testing.T) {
	e := New(Settings{
		Rounds:    2,
		Breathing: 30 * time.Second,
		Recovery:  10 * time.Second,
		Mode:      BreathingModeTimed,
	})

	// Add 30s to round 1 breathing
	ok := e.AddBreathingTime(30 * time.Second)
	if !ok {
		t.Fatal("expected AddBreathingTime to succeed in PhaseBreathing")
	}
	if e.PhaseClock() != 60*time.Second {
		t.Fatalf("expected clock to be 60s, got %s", e.PhaseClock())
	}

	// Advance through round 1
	e.Tick(60 * time.Second)
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected PhaseRetention after 60s, got %s", e.Phase())
	}
	e.Advance()              // to recovery
	e.Tick(10 * time.Second) // finishes round 1
	e.Advance()              // to round 2 breathing

	// Verify bonus time does NOT affect round 2!
	if e.Round() != 2 {
		t.Fatalf("expected round 2, got %d", e.Round())
	}
	if e.BonusBreathing() != 0 {
		t.Fatalf("expected bonusBreathing to be 0 for round 2, got %s", e.BonusBreathing())
	}
	if e.PhaseClock() != 30*time.Second {
		t.Fatalf("expected round 2 clock to be 30s, got %s", e.PhaseClock())
	}
}

func TestCountedBreathingMode(t *testing.T) {
	e := New(Settings{
		Rounds:        2,
		Breathing:     1 * time.Minute,
		Recovery:      10 * time.Second,
		Mode:          BreathingModeCounted,
		TargetBreaths: 5,
	})

	if e.Mode() != BreathingModeCounted {
		t.Fatalf("expected counted mode, got %s", e.Mode())
	}
	if e.TargetBreaths() != 5 {
		t.Fatalf("expected target 5, got %d", e.TargetBreaths())
	}

	// Time counts up in counted mode
	e.Tick(10 * time.Second)
	if e.PhaseElapsed() != 10*time.Second || e.PhaseClock() != 10*time.Second {
		t.Fatalf("expected elapsed 10s, got %s", e.PhaseClock())
	}

	// Increment breaths
	for i := 1; i <= 4; i++ {
		ok := e.IncrementBreaths()
		if !ok || e.Breaths() != i {
			t.Fatalf("expected breath %d, got %d", i, e.Breaths())
		}
	}
	if e.Phase() != PhaseBreathing {
		t.Fatalf("expected still in PhaseBreathing at 4 breaths, got %s", e.Phase())
	}
	if p := e.Progress(); p != 4.0/5.0 {
		t.Fatalf("expected progress 0.8, got %f", p)
	}

	// 5th breath reaches target -> auto to retention!
	e.IncrementBreaths()
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected PhaseRetention after reaching target breaths, got %s", e.Phase())
	}

	// Advance to recovery and round 2
	e.Advance()              // recovery
	e.Tick(10 * time.Second) // finishes round 1
	e.Advance()              // start round 2

	if e.Round() != 2 {
		t.Fatalf("expected round 2, got %d", e.Round())
	}
	if e.Breaths() != 0 {
		t.Fatalf("expected breaths reset to 0 for round 2, got %d", e.Breaths())
	}
}

func TestDecrementBreaths(t *testing.T) {
	e := New(Settings{
		Rounds:        1,
		Breathing:     1 * time.Minute,
		Recovery:      10 * time.Second,
		Mode:          BreathingModeCounted,
		TargetBreaths: 10,
	})

	// Decrement when 0 returns false and stays 0
	if e.DecrementBreaths() {
		t.Fatal("expected DecrementBreaths to return false at 0")
	}
	if e.Breaths() != 0 {
		t.Fatalf("expected 0 breaths, got %d", e.Breaths())
	}

	// Increment to 3
	e.IncrementBreaths()
	e.IncrementBreaths()
	e.IncrementBreaths()
	if e.Breaths() != 3 {
		t.Fatalf("expected 3 breaths, got %d", e.Breaths())
	}

	// Decrement back to 2
	if !e.DecrementBreaths() {
		t.Fatal("expected DecrementBreaths to succeed")
	}
	if e.Breaths() != 2 {
		t.Fatalf("expected 2 breaths after decrement, got %d", e.Breaths())
	}
}

func TestRewindRetentionMidFlight(t *testing.T) {
	e := New(Settings{
		Rounds:    2,
		Breathing: 1 * time.Minute,
		Recovery:  30 * time.Second,
	})

	// Advance through breathing to retention
	e.Advance()
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected retention, got %s", e.Phase())
	}

	// 10 seconds into retention
	e.Tick(10 * time.Second)
	if e.PhaseClock() != 10*time.Second {
		t.Fatalf("expected 10s retention, got %s", e.PhaseClock())
	}

	// Accidental double-enter: skipped retention into recovery, then into next round / ready
	e.Advance() // into recovery
	if e.Phase() != PhaseRecovery {
		t.Fatalf("expected recovery, got %s", e.Phase())
	}
	e.Advance() // finishes round 1, ready for round 2
	if e.Phase() != PhaseRoundReady {
		t.Fatalf("expected round ready, got %s", e.Phase())
	}

	// User realizes mistake and presses Back
	rewoundRound, ok := e.Rewind()
	if !ok || !rewoundRound {
		t.Fatalf("expected Rewind to succeed and rewind round, got ok=%v rewoundRound=%v", ok, rewoundRound)
	}
	if e.Phase() != PhaseRecovery {
		t.Fatalf("expected recovery phase after first rewind, got %s", e.Phase())
	}

	// User presses Back again
	rewoundRound, ok = e.Rewind()
	if !ok || rewoundRound {
		t.Fatalf("expected Rewind to succeed within same round, got ok=%v rewoundRound=%v", ok, rewoundRound)
	}
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected retention phase after second rewind, got %s", e.Phase())
	}

	// Crucial check: retention clock must be restored to exactly 10s!
	if e.PhaseClock() != 10*time.Second {
		t.Fatalf("expected restored retention clock to be 10s, got %s", e.PhaseClock())
	}

	// User continues holding breath for another 15 seconds (total 25s)
	e.Tick(15 * time.Second)
	if e.PhaseClock() != 25*time.Second {
		t.Fatalf("expected 25s retention, got %s", e.PhaseClock())
	}

	// Now advance properly
	e.Advance()              // into recovery
	e.Tick(30 * time.Second) // recovery complete

	results := e.Results()
	if len(results) != 1 || results[0].Retention != 25*time.Second {
		t.Fatalf("expected 1 round with 25s retention, got %#v", results)
	}
}

func TestRewindCompletedPhaseRestartsAtZero(t *testing.T) {
	e := New(Settings{
		Rounds:    1,
		Breathing: 3 * time.Second,
		Recovery:  2 * time.Second,
	})

	// Breathing countdown completes naturally
	e.Tick(3 * time.Second)
	if e.Phase() != PhaseRetention {
		t.Fatalf("expected retention after 3s, got %s", e.Phase())
	}

	// User rewinds back to breathing
	rewoundRound, ok := e.Rewind()
	if !ok {
		t.Fatal("expected Rewind to succeed")
	}
	if rewoundRound {
		t.Fatal("expected rewoundRound to be false within round 1")
	}
	if e.Phase() != PhaseBreathing {
		t.Fatalf("expected PhaseBreathing, got %s", e.Phase())
	}

	// Since breathing completed naturally before, rewinding must start it from 0
	if e.PhaseElapsed() != 0 {
		t.Fatalf("expected phaseElapsed 0, got %s", e.PhaseElapsed())
	}
	if e.PhaseClock() != 3*time.Second {
		t.Fatalf("expected full 3s remaining on clock, got %s", e.PhaseClock())
	}
	if e.SessionElapsed() != 0 {
		t.Fatalf("expected sessionElapsed 0, got %s", e.SessionElapsed())
	}
}

func TestRewindAcrossRounds(t *testing.T) {
	e := New(Settings{
		Rounds:        2,
		Breathing:     2 * time.Second,
		Recovery:      2 * time.Second,
		AutoNextRound: true,
	})

	e.Tick(2 * time.Second) // finish breathing R1
	e.Advance()             // skip retention R1
	e.Tick(2 * time.Second) // finish recovery R1 -> auto starts R2

	if e.Round() != 2 || e.Phase() != PhaseBreathing {
		t.Fatalf("expected Round 2 Breathing, got Round %d Phase %s", e.Round(), e.Phase())
	}
	if len(e.Results()) != 1 {
		t.Fatalf("expected 1 completed round result, got %d", len(e.Results()))
	}

	// Rewind back to Round 1
	rewoundRound, ok := e.Rewind()
	if !ok || !rewoundRound {
		t.Fatalf("expected rewoundRound=true, got ok=%v, rewoundRound=%v", ok, rewoundRound)
	}
	if e.Round() != 1 || e.Phase() != PhaseRecovery {
		t.Fatalf("expected Round 1 Recovery, got Round %d Phase %s", e.Round(), e.Phase())
	}
	// Result for Round 1 should be removed
	if len(e.Results()) != 0 {
		t.Fatalf("expected 0 completed round results after rewind, got %d", len(e.Results()))
	}

	events := e.PopEvents()
	foundRewoundEvent := false
	for _, ev := range events {
		if ev.Type == EventRoundRewound {
			foundRewoundEvent = true
			break
		}
	}
	if !foundRewoundEvent {
		t.Fatal("expected EventRoundRewound event")
	}
}

func TestResetCurrentRound(t *testing.T) {
	e := New(Settings{
		Rounds:    2,
		Breathing: 10 * time.Second,
		Recovery:  5 * time.Second,
	})

	e.Tick(4 * time.Second) // 4s breathing
	e.Advance()             // to retention
	e.Tick(8 * time.Second) // 8s retention

	if e.SessionElapsed() != 12*time.Second {
		t.Fatalf("expected sessionElapsed 12s, got %s", e.SessionElapsed())
	}

	// Reset round
	if !e.ResetCurrentRound() {
		t.Fatal("expected ResetCurrentRound to return true")
	}

	if e.Phase() != PhaseBreathing {
		t.Fatalf("expected PhaseBreathing, got %s", e.Phase())
	}
	if e.PhaseElapsed() != 0 {
		t.Fatalf("expected phaseElapsed 0, got %s", e.PhaseElapsed())
	}
	if e.SessionElapsed() != 0 {
		t.Fatalf("expected sessionElapsed 0 after round reset, got %s", e.SessionElapsed())
	}

	// Rewind undoes the round reset!
	_, ok := e.Rewind()
	if !ok {
		t.Fatal("expected Rewind to undo round reset")
	}
	if e.Phase() != PhaseRetention || e.PhaseClock() != 8*time.Second {
		t.Fatalf("expected restored PhaseRetention with 8s clock, got %s with %s", e.Phase(), e.PhaseClock())
	}
}
