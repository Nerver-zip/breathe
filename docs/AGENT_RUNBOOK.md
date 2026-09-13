# Agent runbook

## Recommended single-run flow

1. Set the repository goal using the exact `/goal` command in `scripts/goal.txt`.
2. Let the agent read `AGENTS.md` and the docs before changing code.
3. The agent should execute all milestones without waiting for intermediate approval.
4. It should continuously run focused tests while implementing and finish with `make check`.
5. Its final response should list implemented features, validation evidence, any intentionally deferred non-MVP item, and the two short-duration manual smoke commands.

## If the agent loses context

Point it back to `GOAL.md` and `docs/ACCEPTANCE.md`; do not restate the product from memory. Those files are written to be self-contained.

## Definition of done

The repository is done only when the acceptance checklist is satisfied and the app can complete a two-round, seconds-long smoke session interactively with correct persisted retention stats.
