# MVP acceptance checklist

The agent should treat every unchecked item as work remaining.

## Build and quality

- [x] `gofmt` leaves no changes.
- [x] `go test ./...` passes.
- [x] `go vet ./...` passes.
- [x] `go build ./...` passes.
- [x] CI runs equivalent checks.

## Session behavior

- [x] Default session is 3 rounds, 3m breathing, 30s recovery.
- [x] Breathing clock counts down and automatically enters retention at zero.
- [x] Retention clock starts at 00:00 and counts upward indefinitely.
- [x] User can end retention with Enter/next.
- [x] Recovery counts down and finalizes the round at zero.
- [x] Manual next-round mode waits after recovery.
- [x] Auto-next mode immediately starts the next breathing phase.
- [x] Final recovery enters a completion summary, not another round.
- [x] Phase timer and session-active timer are both visible.
- [x] Pause freezes active timers and resume does not jump forward.
- [x] Large/delayed Tea ticks do not create timer drift or negative countdowns.
- [x] Active quit opens a confirmation overlay.
- [x] Confirmed abandonment preserves completed rounds without counting the incomplete current round.

## Data and stats

- [x] SQLite data path respects XDG conventions.
- [x] Sessions and each completed round are persisted reliably.
- [x] Per-round retention is stored.
- [x] Today session count is correct.
- [x] Today completed-round count is correct.
- [x] All-time sessions/rounds/active time are correct.
- [x] Average, best, and latest retention are correct.
- [x] Current and best activity streak are correct.
- [x] 7-day daily series fills missing dates with zero.
- [x] ~4-month heatmap groups by local date.
- [x] Abandoned sessions do not create false completed rounds/streak days.

## Stats UI

- [x] `breath stats` opens a full-screen dashboard.
- [x] Dashboard has Today, Retention, Streak, 7-day chart, heatmap, All-time.
- [x] Empty database has a friendly zero-data state.
- [x] Stats UI remains usable at ~80x24.
- [x] `q` exits and `?` exposes help.
- [x] `breath stats --plain` works for terminals/scripts.

## Config and personalization

- [x] Config file is created with valid defaults.
- [x] `breath config show` and `path` work.
- [x] `breath config set <key> <value>` validates and persists supported settings.
- [x] CLI start flags override config for one run.
- [x] At least 7 built-in themes exist: default, catppuccin-mocha, dracula, gruvbox, nord, tokyo-night, solarized.
- [x] `theme list`, `theme set`, `theme preview` work.
- [x] Unknown theme/settings fail with actionable errors.

## Notifications and robustness

- [x] Notifications/bell are configurable.
- [x] Unsupported notification systems fail gracefully.
- [x] DB/config errors are surfaced without corrupting history.
- [x] No network connection is required at runtime.

## Safety and scope

- [x] README/help retains the fainting/water/driving safety warning.
- [x] No medical or physiological interpretation of retention times is presented.
- [x] No cloud, accounts, telemetry, or web service is introduced.
