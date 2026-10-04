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

type BreathingMode string

const (
	BreathingModeTimed   BreathingMode = "timed"
	BreathingModeCounted BreathingMode = "counted"
)

type Settings struct {
	Rounds        int
	Breathing     time.Duration
	Recovery      time.Duration
	AutoNextRound bool
	Mode          BreathingMode
	TargetBreaths int
}

type RoundResult struct {
	Index     int
	Breathing time.Duration
	Retention time.Duration
	Recovery  time.Duration
	Breaths   int
}

type EventType string

const (
	EventBreathingComplete EventType = "breathing_complete"
	EventRecoveryComplete  EventType = "recovery_complete"
	EventSessionComplete   EventType = "session_complete"
	EventRoundRewound      EventType = "round_rewound"
)

type TransitionEvent struct {
	Type  EventType
	Round int
	Auto  bool
}

type PhaseSnapshot struct {
	Phase          Phase
	Round          int
	PhaseElapsed   time.Duration
	SessionElapsed time.Duration
	BonusBreathing time.Duration
	Breaths        int
	Current        RoundResult
	Results        []RoundResult
	CompletedFully bool
}

type Engine struct {
	settings       Settings
	phase          Phase
	round          int
	phaseElapsed   time.Duration
	sessionElapsed time.Duration
	bonusBreathing time.Duration
	breaths        int
	paused         bool
	results        []RoundResult
	current        RoundResult
	events         []TransitionEvent
	history        []PhaseSnapshot
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
	if settings.Mode != BreathingModeCounted {
		settings.Mode = BreathingModeTimed
	}
	if settings.TargetBreaths <= 0 {
		settings.TargetBreaths = 30
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
		if e.settings.Mode == BreathingModeTimed {
			totalBreathing := e.settings.Breathing + e.bonusBreathing
			remaining := totalBreathing - e.phaseElapsed
			step := minDuration(delta, remaining)
			e.phaseElapsed += step
			e.sessionElapsed += step
			e.current.Breathing += step
			if e.phaseElapsed >= totalBreathing {
				e.pushSnapshot(true)
				e.toRetention(true)
				if delta > step {
					e.Tick(delta - step)
				}
			}
		} else {
			e.phaseElapsed += delta
			e.sessionElapsed += delta
			e.current.Breathing += delta
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
			e.pushSnapshot(true)
			e.finishRound(true)
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
	e.pushSnapshot(false)
	switch e.phase {
	case PhaseBreathing:
		e.toRetention(false)
	case PhaseRetention:
		e.phase = PhaseRecovery
		e.phaseElapsed = 0
	case PhaseRecovery:
		e.finishRound(false)
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

func (e *Engine) SetPaused(paused bool) {
	if e.phase == PhaseComplete || e.phase == PhaseRoundReady {
		return
	}
	e.paused = paused
}

func (e *Engine) ResetCurrentPhase() bool {
	if e.phase == PhaseComplete || e.phase == PhaseRoundReady {
		return false
	}
	e.pushSnapshot(false)
	switch e.phase {
	case PhaseBreathing:
		e.sessionElapsed = clampNonNegative(e.sessionElapsed - e.current.Breathing)
		e.current.Breathing = 0
		e.phaseElapsed = 0
		e.bonusBreathing = 0
		e.breaths = 0
		e.current.Breaths = 0
		return true
	case PhaseRetention:
		e.sessionElapsed = clampNonNegative(e.sessionElapsed - e.current.Retention)
		e.current.Retention = 0
		e.phaseElapsed = 0
		return true
	case PhaseRecovery:
		e.sessionElapsed = clampNonNegative(e.sessionElapsed - e.current.Recovery)
		e.current.Recovery = 0
		e.phaseElapsed = 0
		return true
	}
	return false
}

func (e *Engine) ResetCurrentRound() bool {
	if e.phase == PhaseComplete {
		return false
	}
	e.pushSnapshot(false)

	// If in PhaseRoundReady, the round was already completed and appended to results
	if e.phase == PhaseRoundReady && len(e.results) > 0 {
		e.results = e.results[:len(e.results)-1]
		e.events = append(e.events, TransitionEvent{
			Type:  EventRoundRewound,
			Round: e.round,
		})
	}

	// Active time accrued in the current round
	roundElapsed := e.current.Breathing + e.current.Retention + e.current.Recovery
	e.sessionElapsed = clampNonNegative(e.sessionElapsed - roundElapsed)

	e.current = RoundResult{Index: e.round}
	e.phase = PhaseBreathing
	e.phaseElapsed = 0
	e.bonusBreathing = 0
	e.breaths = 0

	return true
}

func (e *Engine) pushSnapshot(completedFully bool) {
	snap := PhaseSnapshot{
		Phase:          e.phase,
		Round:          e.round,
		PhaseElapsed:   e.phaseElapsed,
		SessionElapsed: e.sessionElapsed,
		BonusBreathing: e.bonusBreathing,
		Breaths:        e.breaths,
		Current:        e.current,
		Results:        make([]RoundResult, len(e.results)),
		CompletedFully: completedFully,
	}
	copy(snap.Results, e.results)
	e.history = append(e.history, snap)
	if len(e.history) > 100 {
		e.history = e.history[len(e.history)-100:]
	}
}

func (e *Engine) CanRewind() bool {
	return len(e.history) > 0
}

func (e *Engine) Rewind() (bool, bool) {
	if len(e.history) == 0 {
		return false, false
	}
	lastIdx := len(e.history) - 1
	snap := e.history[lastIdx]
	e.history = e.history[:lastIdx]

	oldRound := e.round
	prevResultsLen := len(e.results)

	e.phase = snap.Phase
	e.round = snap.Round
	e.bonusBreathing = snap.BonusBreathing
	e.breaths = snap.Breaths
	e.current = snap.Current
	e.results = make([]RoundResult, len(snap.Results))
	copy(e.results, snap.Results)

	if snap.CompletedFully {
		// Completed naturally: restart this phase from 0
		e.sessionElapsed = clampNonNegative(snap.SessionElapsed - snap.PhaseElapsed)
		e.phaseElapsed = 0
		e.breaths = 0
		e.current.Breaths = 0
		switch snap.Phase {
		case PhaseBreathing:
			e.current.Breathing = 0
		case PhaseRetention:
			e.current.Retention = 0
		case PhaseRecovery:
			e.current.Recovery = 0
		}
	} else {
		// Interrupted/skipped mid-flight: restore exact elapsed duration
		e.phaseElapsed = snap.PhaseElapsed
		e.sessionElapsed = snap.SessionElapsed
	}

	rewoundRound := false
	if oldRound > snap.Round || len(snap.Results) < prevResultsLen {
		rewoundRound = true
		e.events = append(e.events, TransitionEvent{
			Type:  EventRoundRewound,
			Round: oldRound,
		})
	}

	return rewoundRound, true
}

func (e *Engine) PopEvents() []TransitionEvent {
	if len(e.events) == 0 {
		return nil
	}
	evs := e.events
	e.events = nil
	return evs
}

func (e *Engine) toRetention(auto bool) {
	e.phase = PhaseRetention
	e.phaseElapsed = 0
	if auto {
		e.events = append(e.events, TransitionEvent{
			Type:  EventBreathingComplete,
			Round: e.round,
			Auto:  true,
		})
	}
}

func (e *Engine) finishRound(auto bool) {
	e.results = append(e.results, e.current)
	completedRound := e.round
	e.events = append(e.events, TransitionEvent{
		Type:  EventRecoveryComplete,
		Round: completedRound,
		Auto:  auto,
	})
	if e.round >= e.settings.Rounds {
		e.phase = PhaseComplete
		e.phaseElapsed = 0
		e.events = append(e.events, TransitionEvent{
			Type:  EventSessionComplete,
			Round: completedRound,
			Auto:  auto,
		})
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
	e.bonusBreathing = 0
	e.breaths = 0
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
func (e *Engine) Settings() Settings               { return e.settings }
func (e *Engine) CurrentRound() RoundResult        { return e.current }
func (e *Engine) Mode() BreathingMode {
	if e.settings.Mode == "" {
		return BreathingModeTimed
	}
	return e.settings.Mode
}
func (e *Engine) TargetBreaths() int {
	if e.settings.TargetBreaths <= 0 {
		return 30
	}
	return e.settings.TargetBreaths
}
func (e *Engine) Breaths() int {
	return e.breaths
}
func (e *Engine) BonusBreathing() time.Duration {
	return e.bonusBreathing
}

// AddBreathingTime adds extra time to the current breathing countdown.
// Only applies to the active round and does not alter subsequent rounds.
func (e *Engine) AddBreathingTime(d time.Duration) bool {
	if e.phase != PhaseBreathing || e.settings.Mode != BreathingModeTimed {
		return false
	}
	if d <= 0 {
		return false
	}
	e.bonusBreathing += d
	return true
}

// IncrementBreaths records one breath taken in counted breathing mode.
// Automatically transitions to retention once the target is met.
func (e *Engine) IncrementBreaths() bool {
	if e.phase != PhaseBreathing || e.settings.Mode != BreathingModeCounted {
		return false
	}
	e.breaths++
	e.current.Breaths = e.breaths
	if e.breaths >= e.settings.TargetBreaths {
		e.pushSnapshot(true)
		e.toRetention(true)
	}
	return true
}

// DecrementBreaths subtracts one breath from the counter in case of mistake.
// Cannot go below 0.
func (e *Engine) DecrementBreaths() bool {
	if e.phase != PhaseBreathing || e.settings.Mode != BreathingModeCounted {
		return false
	}
	if e.breaths > 0 {
		e.breaths--
		e.current.Breaths = e.breaths
		return true
	}
	return false
}

func (e *Engine) PhaseClock() time.Duration {
	switch e.phase {
	case PhaseBreathing:
		if e.settings.Mode == BreathingModeTimed {
			return clampNonNegative((e.settings.Breathing + e.bonusBreathing) - e.phaseElapsed)
		}
		return e.phaseElapsed
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
		if e.settings.Mode == BreathingModeTimed {
			return fraction(e.phaseElapsed, e.settings.Breathing+e.bonusBreathing)
		}
		if e.settings.TargetBreaths <= 0 {
			return 0
		}
		v := float64(e.breaths) / float64(e.settings.TargetBreaths)
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
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
