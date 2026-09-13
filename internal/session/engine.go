package session

import "time"

type Phase string

const (
	PhaseBreathing  Phase = "breathing"
	PhaseRetention  Phase = "retention"
	PhaseRecovery   Phase = "recovery"
	PhaseRoundReady Phase = "round_ready"
	PhaseComplete   Phase = "complete"
)

type Settings struct {
	Rounds        int
	Breathing     time.Duration
	Recovery      time.Duration
	AutoNextRound bool
}

type RoundResult struct {
	Index     int
	Breathing time.Duration
	Retention time.Duration
	Recovery  time.Duration
}

type Engine struct {
	settings       Settings
	phase          Phase
	round          int
	phaseElapsed   time.Duration
	sessionElapsed time.Duration
	paused         bool
	results        []RoundResult
	current        RoundResult
}

func New(settings Settings) *Engine {
	if settings.Rounds < 1 {
		settings.Rounds = 1
	}
	if settings.Breathing <= 0 {
		settings.Breathing = 3 * time.Minute
	}
	if settings.Recovery <= 0 {
		settings.Recovery = 30 * time.Second
	}
	return &Engine{
		settings: settings,
		phase:    PhaseBreathing,
		round:    1,
		current:  RoundResult{Index: 1},
	}
}

func (e *Engine) Tick(delta time.Duration) {
	if delta <= 0 || e.paused || e.phase == PhaseComplete || e.phase == PhaseRoundReady {
		return
	}

	switch e.phase {
	case PhaseBreathing:
		remaining := e.settings.Breathing - e.phaseElapsed
		step := minDuration(delta, remaining)
		e.phaseElapsed += step
		e.sessionElapsed += step
		e.current.Breathing += step
		if e.phaseElapsed >= e.settings.Breathing {
			e.toRetention()
			if delta > step {
				e.Tick(delta - step)
			}
		}
	case PhaseRetention:
		e.phaseElapsed += delta
		e.sessionElapsed += delta
		e.current.Retention += delta
	case PhaseRecovery:
		remaining := e.settings.Recovery - e.phaseElapsed
		step := minDuration(delta, remaining)
		e.phaseElapsed += step
		e.sessionElapsed += step
		e.current.Recovery += step
		if e.phaseElapsed >= e.settings.Recovery {
			e.finishRound()
			if delta > step && e.phase == PhaseBreathing {
				e.Tick(delta - step)
			}
		}
	}
}

func (e *Engine) Advance() {
	if e.phase == PhaseComplete {
		return
	}
	switch e.phase {
	case PhaseBreathing:
		e.toRetention()
	case PhaseRetention:
		e.phase = PhaseRecovery
		e.phaseElapsed = 0
	case PhaseRecovery:
		e.finishRound()
	case PhaseRoundReady:
		e.startNextRound()
	}
}

func (e *Engine) TogglePause() {
	if e.phase == PhaseComplete || e.phase == PhaseRoundReady {
		return
	}
	e.paused = !e.paused
}

func (e *Engine) toRetention() {
	e.phase = PhaseRetention
	e.phaseElapsed = 0
}

func (e *Engine) finishRound() {
	e.results = append(e.results, e.current)
	if e.round >= e.settings.Rounds {
		e.phase = PhaseComplete
		e.phaseElapsed = 0
		return
	}
	if e.settings.AutoNextRound {
		e.startNextRound()
		return
	}
	e.phase = PhaseRoundReady
	e.phaseElapsed = 0
}

func (e *Engine) startNextRound() {
	e.round++
	e.phase = PhaseBreathing
	e.phaseElapsed = 0
	e.paused = false
	e.current = RoundResult{Index: e.round}
}

func (e *Engine) Phase() Phase                     { return e.phase }
func (e *Engine) Round() int                       { return e.round }
func (e *Engine) TotalRounds() int                 { return e.settings.Rounds }
func (e *Engine) Paused() bool                     { return e.paused }
func (e *Engine) PhaseElapsed() time.Duration      { return e.phaseElapsed }
func (e *Engine) SessionElapsed() time.Duration    { return e.sessionElapsed }
func (e *Engine) BreathingDuration() time.Duration { return e.settings.Breathing }
func (e *Engine) RecoveryDuration() time.Duration  { return e.settings.Recovery }
func (e *Engine) Done() bool                       { return e.phase == PhaseComplete }

func (e *Engine) PhaseClock() time.Duration {
	switch e.phase {
	case PhaseBreathing:
		return clampNonNegative(e.settings.Breathing - e.phaseElapsed)
	case PhaseRetention:
		return e.phaseElapsed
	case PhaseRecovery:
		return clampNonNegative(e.settings.Recovery - e.phaseElapsed)
	default:
		return 0
	}
}

func (e *Engine) Progress() float64 {
	switch e.phase {
	case PhaseBreathing:
		return fraction(e.phaseElapsed, e.settings.Breathing)
	case PhaseRecovery:
		return fraction(e.phaseElapsed, e.settings.Recovery)
	default:
		return 0
	}
}

func (e *Engine) Results() []RoundResult {
	out := make([]RoundResult, len(e.results))
	copy(out, e.results)
	return out
}

func fraction(a, b time.Duration) float64 {
	if b <= 0 {
		return 0
	}
	v := float64(a) / float64(b)
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampNonNegative(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
