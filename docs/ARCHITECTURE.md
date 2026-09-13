# Architecture

## Stack

Use the same broad stack family as the reference Pomodoro TUI:

- Go
- Charm Bubble Tea for event loop/model/update/view
- Bubbles for reusable TUI components where valuable
- Lip Gloss for layout/styles
- Cobra for CLI
- Viper for config
- SQLite for local persistence
- `sqlx` for query ergonomics
- pure-Go `modernc.org/sqlite` driver for easy cross-platform builds
- optional `beeep` for best-effort desktop notifications

## Package boundaries

Preferred final layout:

```text
cmd/                    Cobra commands
config/                 loading, validation, mutation
internal/session/       pure state machine and session domain
internal/storage/       SQLite schema, repositories, migrations/queries
internal/tui/session/   main session Bubble Tea model and components
internal/tui/stats/     stats dashboard model
internal/tui/components reusable chart/heatmap/help/confirm components
internal/theme/         theme registry and preview
internal/notify/        notification abstraction
```

The starter has a flatter `internal/tui` package. Refactor only when it improves clarity; package churn is not itself a goal.

## State machine

```text
BREATHING --timer/advance--> RETENTION --advance--> RECOVERY
RECOVERY --timer/advance--> [COMPLETE if last round]
RECOVERY --timer/advance--> ROUND_READY --advance--> BREATHING(next round)
RECOVERY --timer/advance--> BREATHING(next round)   (auto-next)
```

Orthogonal state: paused/unpaused. `ROUND_READY` and `COMPLETE` are not active exercise phases and should not advance the active session clock.

### Invariants

- retention never finishes by timer;
- round index is 1-based in UI and persisted results;
- completed round count equals the number of finalized `RoundResult`s;
- session active duration is the sum of active phase durations, excluding pause and round-ready waits;
- repeated `Advance`/tick events cannot finalize the same round twice;
- timers remain monotonic and non-negative;
- oversize tick deltas may cross a timed phase boundary without losing elapsed time.

## Persistence model

Starter schema uses two tables:

### sessions

- `id`
- UTC `started_at`, `ended_at`
- `local_date` for stable local-day grouping
- `planned_rounds`
- `completed_rounds`
- `active_duration_ms`
- `wall_duration_ms`
- `status`: `completed` or `abandoned`

### rounds

- `id`
- `session_id`
- `round_index`
- `breathing_ms`
- `retention_ms`
- `recovery_ms`

All duration values are integer milliseconds. If migrations become necessary, introduce a tiny schema-version/migration mechanism rather than destructive resets.

## Persistence timing

The starter writes at TUI exit. The final MVP should be more crash-tolerant:

- create session when it begins;
- persist a round transactionally as soon as it completes;
- finalize session status/end times on normal completion or confirmed abandonment;
- if an unfinished session is found on startup, either mark it abandoned deterministically or offer a narrowly scoped resume flow. Resume is optional; silent corruption is not.

## Stats queries

Use SQL for aggregation, not in-memory scanning of every historical row. Required query families:

- all-time totals;
- current-day totals;
- avg/max/latest retention;
- per-day completed round counts for 7 days and ~4 months;
- current/best streak computed from distinct active local dates.

Make date boundaries explicit and test them with fixed dates.

## TUI architecture

Business state lives in `session.Engine`; rendering reacts to snapshots. The Bubble Tea model owns key mapping, viewport size, overlays, and notification commands. Database access should happen outside `View()` and preferably through Tea commands/messages for the stats screen.

Do not put SQL, config mutation, or wall-clock calculations in rendering functions.

## Time and drift

A periodic Tea tick should carry `time.Time`. Compute `delta = now - lastTick` and advance the engine by `delta`. Never increment counters by a hard-coded 100ms just because the nominal render tick is 100ms.

When paused, update `lastTick` but do not advance active timers, preventing a large catch-up delta on resume.

## Test strategy

- table-driven state-machine tests;
- temporary SQLite databases for repository/query tests;
- config tests with temporary XDG dirs;
- render smoke tests for narrow/normal terminals where practical;
- CLI command tests for validation and machine-readable outputs.
