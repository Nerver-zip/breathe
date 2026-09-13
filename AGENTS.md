# Agent instructions

## Read first

In this order:

1. `GOAL.md`
2. `docs/PRODUCT.md`
3. `docs/ARCHITECTURE.md`
4. `docs/IMPLEMENTATION_PLAN.md`
5. `docs/ACCEPTANCE.md`
6. `docs/REFERENCE_POMO.md`
7. `docs/SCAFFOLD_STATUS.md`
8. current source and tests

## Working rules

- Treat `GOAL.md` and `docs/ACCEPTANCE.md` as binding.
- Prefer pure, testable state transitions over timer logic embedded in rendering code.
- Use real elapsed deltas from `time.Time`; do not assume a Bubble Tea tick arrives exactly on schedule.
- Keep pause semantics consistent: active exercise clocks stop while paused; wall-clock duration may continue and is persisted separately.
- The retention phase is intentionally open-ended and must never auto-finish.
- Timed breathing automatically enters retention at zero.
- Recovery automatically completes at zero. Whether the next round starts immediately is controlled by `auto_next_round`.
- Count a round as completed only once recovery finishes or is explicitly advanced through.
- Store durations as integer milliseconds. Store timestamps in UTC and the derived local date used for daily statistics.
- Preserve local-first behavior and XDG-friendly paths.
- Keep rendering responsive down to roughly 80x24 and graceful below that.
- Avoid medical claims. Preserve the safety warning.
- No network calls in app runtime.
- Do not copy code from the reference project; reproduce patterns independently.

## Quality gate

Run `make check` plus targeted smoke tests. Add tests whenever changing state transitions, persistence, date grouping, streaks, config parsing, or CLI behavior.
