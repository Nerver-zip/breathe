# One-run implementation plan

The goal is deliberately sequenced so an autonomous agent can reach a coherent MVP without stopping for design approval.

## Milestone 0 — preflight

Read all agent/product docs and run the starter quality gate. Record gaps, but do not rewrite the architecture unless a blocker is proven.

Commands:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./...
```

## Milestone 1 — harden the session engine

- Cover all transitions with table-driven tests.
- Verify oversize tick deltas crossing timed phase boundaries.
- Define/reset-current-phase behavior.
- Expose a snapshot/event mechanism if needed for notifications/persistence without coupling Bubble Tea to internals.
- Ensure pause/resume has no catch-up drift.

Exit criterion: deterministic tests cover happy path, manual advance, auto-next, pause, early advance, final round, and large delta.

## Milestone 2 — persistence lifecycle

- Add migration/version support if schema changes.
- Persist session start, completed rounds incrementally, and final status.
- Preserve completed rounds on confirmed abandonment.
- Add repository methods for today/all-time/retention/daily-series/streak stats.
- Add temporary-DB tests, including empty DB and multi-day fixtures.

Exit criterion: no stats calculation depends on the live TUI model.

## Milestone 3 — main TUI polish

- Split reusable styles/components where useful.
- Keep large central phase timer and smaller session timer.
- Add responsive layout and compact fallback.
- Add help overlay (`?`).
- Add quit confirmation overlay.
- Add reset confirmation where destructive.
- Add clear ready-between-rounds state.
- Add final summary with per-round retention, avg, best.
- Make status/progress semantics visually clear without relying on color alone.

Exit criterion: all session states are legible at 80x24 and normal desktop terminal sizes.

## Milestone 4 — notifications

- Introduce a notifier interface with no-op fallback.
- Best-effort desktop notification and terminal bell controlled by config.
- Trigger only on automatic timed transitions and session completion.
- Notification failures must be swallowed/logged, never fatal.

## Milestone 5 — stats dashboard

Implement `breath stats` as a Bubble Tea dashboard inspired by the reference project's composition, but with breathing-specific metrics:

- Today card/summary;
- retention metrics;
- current/best streak;
- 7-day rounds bar chart;
- ~4-month activity heatmap;
- all-time totals;
- help/quit keys;
- useful zero-data state.

Also provide `breath stats --plain`; add `--json` if cleanly achievable.

Exit criterion: dashboard renders with empty data and fixture-rich data without panics.

## Milestone 6 — configuration and themes

- Complete config validation and XDG handling.
- Implement `config set` with typed parsing and atomic write.
- Implement `theme list`, `theme set`, and `theme preview`.
- Add seven built-in themes listed in product spec.
- Keep all style colors behind a theme registry.

## Milestone 7 — UX safety and docs

- Keep a concise safety warning in README and help/about output.
- Do not add health interpretations.
- Ensure active-session quit requires confirmation.
- Clarify whether partial sessions count in each stat: only completed rounds drive streak/day activity.

## Milestone 8 — packaging and CI

- CI on Linux: fmt check, tests, vet, build.
- Optional cross-platform compile matrix if dependencies allow it cleanly.
- Add version injection variables only if useful.
- Keep release tooling lightweight; GoReleaser config is optional for MVP.

## Milestone 9 — final validation

Run:

```bash
make check
go run . config show
go run . stats --plain   # after implementing the flag
```

Manual TUI smoke test with seconds instead of minutes:

```bash
go run . start --rounds 2 --breathing 5s --recovery 3s
go run . start --rounds 2 --breathing 5s --recovery 3s --auto-next
```

Verify pause, retention count-up, manual retention end, recovery, manual/auto round handoff, completion summary, quit confirmation, and persisted stats.
