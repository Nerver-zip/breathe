# Breathing TUI

A local-first terminal application for guided Wim Hof-style breathing sessions: timed breathing, open-ended breath retention, recovery hold, round tracking, and daily statistics.

> [!WARNING]
> Breath-hold exercises can cause dizziness or loss of consciousness. Use this only in a safe seated/lying position, never in or near water, never while driving, and stop if you feel unwell. This project is a timer/tracker, not medical guidance.

## MVP flow

Each round has three phases:

1. **Deep breathing** — countdown, default `03:00 → 00:00`.
2. **Retention** — count-up, `00:00 → ...`, stopped manually when the user needs to breathe.
3. **Recovery hold** — inhale once and hold; countdown, default `00:30 → 00:00`.

After recovery, the next round can start automatically or wait for explicit confirmation. The main timer always represents the current phase; a smaller timer tracks active exercise time across the whole session. Pausing freezes both active timers.

## Starter implementation

This repository is intentionally an agent-ready scaffold, not the final polished MVP. It already contains:

- Go/Cobra CLI.
- Bubble Tea + Lip Gloss full-screen timer.
- Tested phase state machine.
- Pause / advance / quit controls.
- Manual or automatic round transitions.
- SQLite persistence for sessions and round retention times.
- Basic `breath stats` output.
- Default and Catppuccin Mocha theme primitives.
- Detailed product, architecture, implementation, and acceptance docs.
- A one-run Codex `/goal` prompt in [`GOAL.md`](GOAL.md) and [`scripts/goal.txt`](scripts/goal.txt).

The agent's job is to finish the dashboard, configuration UX, notifications, session-resume/quit confirmation, richer themes, tests, and release polish defined in the docs.

## Quick start

```bash
go mod tidy
go run . start
```

Fast manual test:

```bash
go run . start --rounds 2 --breathing 5s --recovery 3s
```

Auto-start next rounds:

```bash
go run . start --rounds 3 --auto-next
```

Show stats:

```bash
go run . stats
```

Configuration is created at `~/.config/breath/config.yaml`. Data is stored under the user's XDG data directory (normally `~/.local/share/breath/breath.db` on Linux).

## Controls

| Key | Action |
|---|---|
| `Space` / `p` | Pause / resume |
| `Enter` / `n` | Advance phase / start next round |
| `q` / `Ctrl+C` | Quit |

The final MVP must add safe quit confirmation while a session is active; see `docs/ACCEPTANCE.md`.

## Agent execution

Open the repository in Codex and paste the command from `scripts/goal.txt`. The goal is explicitly designed for a single autonomous run with validation before completion.

## Inspiration

The UI/product architecture is inspired by [Bahaaio/pomo](https://github.com/Bahaaio/pomo): a Go terminal Pomodoro app using Bubble Tea, Lip Gloss, Cobra/Viper, SQLite, configurable themes, and statistics dashboards. This repository does **not** copy its business logic; the timer state machine and data model are specific to breathing rounds.

## Status

**Scaffold / pre-MVP.** The source compiles and core state transitions are tested. `GOAL.md` defines the finished MVP.
