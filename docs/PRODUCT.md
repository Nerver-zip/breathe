# Product specification

## 1. Product intent

Breathing TUI is a fast keyboard-first terminal timer for multi-round breathing practice. The interaction should feel as immediate and polished as a mature Pomodoro TUI, while the domain model is purpose-built around breathing rounds and retention tracking.

The MVP optimizes for one person on one machine. Everything is stored locally.

## 2. Core session model

A session has `N >= 1` rounds. Defaults:

- rounds: `3`
- breathing phase: `3m`
- retention phase: open-ended
- recovery hold: `30s`
- auto-next-round: `false`

### Phase A — deep breathing

- Central clock counts down from configured duration to `00:00`.
- Instruction: inhale fully, exhale without forcing.
- At zero, transition automatically to retention.
- User may manually advance early.

### Phase B — retention

- Central clock starts at `00:00` and counts upward.
- No automatic timeout.
- `Enter`/configured advance key ends retention and begins recovery.
- The exact active retention duration is recorded.

### Phase C — recovery hold

- Central clock counts down from configured duration.
- Instruction: inhale once and hold.
- At zero the round is complete.
- If this is the final round, show session summary.
- Otherwise:
  - `auto_next_round=true`: immediately start the next breathing phase;
  - `false`: show a ready screen and wait for the user.

## 3. Timer semantics

The UI has two clocks:

- **phase clock**: large and central; countdown for timed phases, count-up for retention;
- **session active clock**: smaller; sum of active breathing + retention + recovery time across the session.

Pausing freezes both active clocks. The database also stores wall-clock session duration separately so the implementation never conflates elapsed real time with active exercise time.

Bubble Tea ticks are rendering opportunities, not the source of truth. Calculate elapsed time from actual timestamps/deltas to avoid drift when the terminal is busy.

## 4. Primary screen

Visual hierarchy:

1. app title / compact brand;
2. `Round X/N` and small session clock;
3. current status (`RUNNING`, `PAUSED`, `ROUND COMPLETE`);
4. phase name and one-line instruction;
5. large ASCII phase timer;
6. progress bar for timed phases or open-ended-retention hint;
7. compact key help.

Desired feel: minimal, centered, low-noise, good spacing, color used semantically. Do not overload the main session view with historical stats.

## 5. Controls

MVP bindings:

- `Space` or `p`: pause/resume;
- `Enter` or `n`: advance phase / confirm next round;
- `q`: request quit;
- `?`: toggle expanded help;
- `r`: restart current phase, with confirmation if it would discard a non-zero retention;
- `s`: open session stats/summary only when not interfering with an active phase (optional shortcut; CLI `stats` is mandatory).

Quitting an active session must show a confirmation overlay. If confirmed, save the session with `status=abandoned` and keep already completed rounds. Partial current-round retention may be shown in the summary but must not count as a completed round.

## 6. Completion summary

After the final recovery:

- total active time;
- completed rounds;
- each round's retention time;
- average retention;
- best retention;
- optional delta from previous round;
- `Enter`/`q` exits, and a key may open the stats dashboard.

## 7. Statistics dashboard

The dashboard should borrow the information density and compositional philosophy of `pomo stats`, adapted to breathing rather than Pomodoro work/break ratios.

Required sections:

- **Today**: sessions completed, rounds completed, active time, best retention;
- **Retention**: all-time average, best, latest; optionally a compact recent-trend sparkline;
- **Streak**: current and best streak, where an active day means at least one completed round;
- **7-day chart**: completed rounds per day (primary) and optionally active minutes as secondary text;
- **~4-month heatmap**: GitHub-style cells by completed rounds per local day;
- **All time**: total sessions, rounds, active time.

`breath stats` opens the TUI dashboard. A `--plain` or `--json` output mode is desirable for scripts; JSON is preferred if time permits but should not displace correctness of the TUI.

## 8. Configuration

Config file path on Linux: `~/.config/breath/config.yaml` (respect `XDG_CONFIG_HOME`).

Data path on Linux: `~/.local/share/breath/breath.db` (respect `XDG_DATA_HOME`).

Settings:

```yaml
rounds: 3
breathing: 3m
recovery: 30s
auto_next_round: false
theme: default
notifications: true
bell: true
```

CLI overrides win for the current run.

Commands:

```bash
breath
breath start
breath start --rounds 4 --breathing 2m30s --recovery 30s --auto-next
breath stats
breath stats --plain
breath config show
breath config path
breath config set rounds 4
breath config set breathing 2m30s
breath theme list
breath theme preview catppuccin-mocha
```

## 9. Themes

Provide a centralized theme model. MVP built-ins should include at least:

- default
- catppuccin-mocha
- dracula
- gruvbox
- nord
- tokyo-night
- solarized

Theme definitions must not leak throughout business logic. A future custom-theme file should be possible without refactoring the session engine.

## 10. Notifications

On timed phase completion, emit a desktop notification if enabled and supported, plus optional terminal bell. Failure to notify must never crash or block the session.

Suggested events:

- breathing complete → retention begins;
- recovery complete / next round ready;
- full session complete.

Do not notify when the user manually advances unless it improves clarity.

## 11. Safety language

The app must display or make readily accessible a concise warning: breath-hold exercises may cause dizziness or fainting; practice seated/lying down, never in water or while driving, and stop if unwell. Do not infer health status from retention data.

## 12. Non-goals for MVP

No account system, cloud sync, multiplayer, health-device integration, breath-rate sensor, automatic breath counting, audio coaching library, mobile UI, web UI, telemetry, or medical interpretation.
