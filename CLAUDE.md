# quinto

Terminal-native web analytics dashboard. Reads a local SQLite file synced from GoatCounter; the same file is queryable by agents via `quinto query`.

Go, no cgo — `modernc.org/sqlite`, so the whole thing cross-compiles to a single binary. Don't introduce a cgo dependency without revisiting `PLAN.md` › Definition of done.

Named for *quinto sabor* — the fifth taste. See `PLAN.md` for scope, resolved decisions, and open items.

## Agent skills

### Issue tracker

Issues live as markdown files under `.scratch/<feature>/` in this repo. See `docs/agents/issue-tracker.md`.

### Triage labels

Default vocabulary, unchanged: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.
