# MVP acceptance checklist

The agent should treat every unchecked item as work remaining.

## Build and quality

- [ ] `gofmt` leaves no changes.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go build ./...` passes.
- [ ] CI runs equivalent checks.

## Session behavior

- [ ] Default session is 3 rounds, 3m breathing, 30s recovery.
- [ ] Breathing clock counts down and automatically enters retention at zero.
- [ ] Retention clock starts at 00:00 and counts upward indefinitely.
- [ ] User can end retention with Enter/next.
- [ ] Recovery counts down and finalizes the round at zero.
- [ ] Manual next-round mode waits after recovery.
- [ ] Auto-next mode immediately starts the next breathing phase.
- [ ] Final recovery enters a completion summary, not another round.
- [ ] Phase timer and session-active timer are both visible.
- [ ] Pause freezes active timers and resume does not jump forward.
- [ ] Large/delayed Tea ticks do not create timer drift or negative countdowns.
- [ ] Active quit opens a confirmation overlay.
- [ ] Confirmed abandonment preserves completed rounds without counting the incomplete current round.

## Data and stats

- [ ] SQLite data path respects XDG conventions.
- [ ] Sessions and each completed round are persisted reliably.
- [ ] Per-round retention is stored.
- [ ] Today session count is correct.
- [ ] Today completed-round count is correct.
- [ ] All-time sessions/rounds/active time are correct.
- [ ] Average, best, and latest retention are correct.
- [ ] Current and best activity streak are correct.
- [ ] 7-day daily series fills missing dates with zero.
- [ ] ~4-month heatmap groups by local date.
- [ ] Abandoned sessions do not create false completed rounds/streak days.

## Stats UI

- [ ] `breath stats` opens a full-screen dashboard.
- [ ] Dashboard has Today, Retention, Streak, 7-day chart, heatmap, All-time.
- [ ] Empty database has a friendly zero-data state.
- [ ] Stats UI remains usable at ~80x24.
- [ ] `q` exits and `?` exposes help.
- [ ] `breath stats --plain` works for terminals/scripts.

## Config and personalization

- [ ] Config file is created with valid defaults.
- [ ] `breath config show` and `path` work.
- [ ] `breath config set <key> <value>` validates and persists supported settings.
- [ ] CLI start flags override config for one run.
- [ ] At least 7 built-in themes exist: default, catppuccin-mocha, dracula, gruvbox, nord, tokyo-night, solarized.
- [ ] `theme list`, `theme set`, `theme preview` work.
- [ ] Unknown theme/settings fail with actionable errors.

## Notifications and robustness

- [ ] Notifications/bell are configurable.
- [ ] Unsupported notification systems fail gracefully.
- [ ] DB/config errors are surfaced without corrupting history.
- [ ] No network connection is required at runtime.

## Safety and scope

- [ ] README/help retains the fainting/water/driving safety warning.
- [ ] No medical or physiological interpretation of retention times is presented.
- [ ] No cloud, accounts, telemetry, or web service is introduced.
