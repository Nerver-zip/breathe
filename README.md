# Breathing TUI

A fast, keyboard-first terminal application for multi-round breathing practice and retention tracking: timed breathing countdown, open-ended breath retention, recovery hold, incremental session persistence, and rich statistics.

> [!WARNING]
> Breath-hold exercises can cause dizziness or loss of consciousness. Practice only in a safe seated or lying position, never in or near water, never while driving or operating machinery, and stop immediately if you feel unwell. This software is a timer and habit tracker, not medical guidance or health assessment.

## Features

- **Guided 3-Phase Rounds:**
  1. **Deep Breathing:** Countdown timer (default `03:00 → 00:00`), guiding deep rhythmic breaths. Auto-transitions to retention at zero.
  2. **Retention:** Count-up timer (`00:00 → ...`), open-ended hold after the exhale. Stopped manually with `Enter` when you need to breathe.
  3. **Recovery Hold:** Countdown timer (default `00:30 → 00:00`), inhale deeply and hold. Auto-completes at zero.
- **Dual Monotonic Clocks:** Central phase clock (ASCII large digits on standard terminals, clean text on compact screens) and session-active duration clock. Pausing freezes active exercise clocks without drift or jump on resume.
- **Incremental Local Persistence:** Sessions and completed rounds are written to local SQLite storage as soon as each round completes. Quitting mid-session preserves completed rounds without false count of the incomplete round.
- **Statistics Dashboard:**
  - **Today:** Sessions completed, rounds completed, active time, best retention.
  - **Retention Metrics:** All-time average, personal best, latest retention duration.
  - **Habit Streak:** Current and all-time best streaks based on active local practice dates.
  - **7-Day Bar Chart:** Daily completed rounds and active exercise duration.
  - **~4-Month Heatmap:** GitHub-style 18-week contribution grid grouped by local date.
  - **All-Time Totals:** Completed sessions, total rounds, cumulative active duration.
- **Command Line & Automation:** Plain text (`--plain`) and JSON (`--json`) output modes for scripting.
- **Personalization & Themes:** 7 built-in themes (`default`, `catppuccin-mocha`, `dracula`, `gruvbox`, `nord`, `tokyo-night`, `solarized`) with live terminal preview.
- **Safe Quit & Reset:** Confirmation overlays protect against accidental abandonment or discarding active retention hold.
- **Notifications & Bell:** Configurable desktop notifications and terminal bell on timed phase completion and session end, with graceful no-op fallback.
- **100% Local-First:** No accounts, telemetry, cloud dependencies, or network calls at runtime. Respects XDG base directories.

## Quick Start

### Installation & Build

```bash
git clone https://github.com/Nerver-zip/breathing-tui.git
cd breathing-tui
make build
# binary is built at bin/breath
```

### Run a Session

Start default session (3 rounds, 3m breathing, 30s recovery):
```bash
./bin/breath start
```

Quick smoke test session with short durations:
```bash
./bin/breath start --rounds 2 --breathing 5s --recovery 3s
```

Auto-advance to the next round immediately after recovery:
```bash
./bin/breath start --rounds 3 --auto-next
```

### View Statistics

Open the interactive full-screen TUI dashboard:
```bash
./bin/breath stats
```

Output for shell scripts:
```bash
./bin/breath stats --plain
./bin/breath stats --json
```

### Configuration

Inspect current effective settings and config path:
```bash
./bin/breath config show
./bin/breath config path
```

Update persistent configuration:
```bash
./bin/breath config set rounds 4
./bin/breath config set breathing 2m30s
./bin/breath config set recovery 30s
./bin/breath config set auto_next_round true
./bin/breath config set theme catppuccin-mocha
./bin/breath config set notifications true
./bin/breath config set bell true
```

Default config file location:
- Linux: `~/.config/breath/config.yaml` (respects `$XDG_CONFIG_HOME`)
- Database: `~/.local/share/breath/breath.db` (respects `$XDG_DATA_HOME`)

### Themes

List available themes:
```bash
./bin/breath theme list
```

Preview a theme with palette swatches and sample components:
```bash
./bin/breath theme preview dracula
./bin/breath theme preview nord
```

Set active theme:
```bash
./bin/breath theme set tokyo-night
```

## Keyboard Controls

| Key | Context | Action |
|---|---|---|
| `Space` / `p` | Active Session | Pause / resume exercise clocks |
| `Enter` / `n` | Active Session | Advance phase / confirm next round |
| `r` | Active Session | Restart current phase (confirms if retention > 0) |
| `?` | Any view | Toggle help & safety overlay |
| `q` / `Esc` | Active Session | Request quit (confirms before abandoning) |
| `Enter` / `q` | Summary Screen | Exit session |
| `s` | Summary Screen | Open full statistics dashboard |
| `q` / `Esc` | Stats / Help | Exit dashboard or dismiss overlay |
| `y` / `n` | Confirmation Modal | Confirm (`y`) or cancel (`n` / `Esc`) |

## Verification & Tests

Run the complete test suite and code quality gate:
```bash
make check
```
This runs formatting checks (`gofmt`), unit and integration tests (`go test ./...`), vet analysis (`go vet ./...`), and compilation (`go build`).

## Safety Disclaimer

This application is purely a timer and session tracker. It does not provide medical guidance, diagnoses, or physiological fitness scores. Never practice breath retention in water, while operating a vehicle, or while standing. Always consult a healthcare professional before beginning vigorous breathwork practices.
