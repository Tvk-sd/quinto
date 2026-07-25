# quinto

Terminal-native web analytics dashboard. Reads a local DuckDB file synced from Umami Cloud; the same file is queryable by agents via `quinto query`.

Named for *quinto sabor* — the fifth taste. See `PLAN.md` for scope, resolved decisions, and open items.

## Agent skills

### Issue tracker

Issues live as markdown files under `.scratch/<feature>/` in this repo. See `docs/agents/issue-tracker.md`.

### Triage labels

Default vocabulary, unchanged: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
