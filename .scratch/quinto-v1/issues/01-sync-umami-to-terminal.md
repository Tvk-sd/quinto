# 01 — Sync one real visit from Umami to the terminal

**What to build:** You visit your own site, run two commands, and see that visit in your terminal. `quinto sync` pulls session and activity data from Umami Cloud into a local DuckDB file; running `quinto` prints the most recent visits as a plain table. No TUI yet — this is the tracer bullet that proves the whole path works end to end.

Running `sync` twice must not duplicate anything or refetch what it already has. A sync command you can't safely re-run is a footgun you'd hit on day one.

The local schema is the spine of everything downstream and is the thing an agent reads, so it should be legible to something that has never seen the codebase. It maps 1:1 onto Umami's responses — no invented vocabulary, no translation layer:

```sql
sessions(session_id PK, first_seen, last_seen, country,
         browser, os, device, referrer)

events(event_id PK, session_id FK, visit_id, created_at,
       url_path, url_query, referrer_domain, event_type, event_name)
```

**Also settle during this ticket:** Umami's sync is naively N+1 — list sessions, then one activity call per session. That's fine at single-digit sessions/day and falls over on a busy site. Check whether Umami Cloud exposes a bulk export endpoint before committing to the N+1 loop. Record what you find.

This is the first ticket in an empty repo, so it also establishes the project skeleton: module setup, dependency choices, config loading, and the testing convention the rest of the tickets inherit.

**Testing policy for this project:** test the sync logic, not the rendering. Idempotency and the incremental watermark are claims you cannot verify by eyeballing a terminal — they need fixtures and assertions. TUI output does not get tests; it gets looked at. This is a two-weekend portfolio project, and that split is where tests actually pay for themselves.

**Blocked by:** An Umami API key — a human prerequisite, tracked in `PLAN.md` › Offen — bei Till. No other tickets.

**Status:** ready-for-agent

- [ ] `quinto sync` authenticates to Umami Cloud with an API key and writes sessions and events into a local DuckDB file
- [ ] Running `quinto sync` twice in a row produces no duplicate rows
- [ ] Sync is incremental — a second run fetches only what is new, tracked by a stored watermark
- [ ] Idempotency and the watermark are covered by tests against fixture responses, not verified by eye
- [ ] `quinto` prints the most recent visits as a readable table
- [ ] A visit you make to your own site appears after a sync
- [ ] The database file lives in a conventional per-user location, and its path is discoverable from the CLI
- [ ] Credentials are read from config or environment, never committed
- [ ] Findings on bulk export vs N+1 are written down
