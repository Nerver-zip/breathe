# Reference project: Bahaaio/pomo

Reference: https://github.com/Bahaaio/pomo

## Patterns intentionally adopted

- Go single-binary CLI/TUI distribution.
- Bubble Tea event-driven model/update/view architecture.
- Lip Gloss styling and centered full-screen presentation.
- Cobra/Viper separation between CLI and configuration.
- SQLite-backed local statistics.
- Distinct stats view built from reusable visual components.
- Weekly activity visualization and GitHub-style multi-month heatmap.
- Theme registry and theme preview/list UX.
- Pause/skip/help keyboard-first controls.
- Best-effort desktop notifications.

## Domain-specific replacements

Pomodoro's work/break cycle becomes the breathing state machine:

```text
breathing countdown -> retention count-up -> recovery countdown
```

The work/break duration ratio is not meaningful here. Replace it with retention summary/trend and round/session counts.

Pomodoro “tasks” are not part of the MVP. Daily activity means completed breathing rounds.

## Clean-room rule

Use the reference to understand patterns and UX. Do not copy source files. Implement this repository independently, preserving its own names, schema, tests, and domain model.
