# 01 — Sync one real visit from GoatCounter to the terminal

**What to build:** You visit your own site, run two commands, and see that visit in your terminal. `quinto sync` pulls individual pageviews from GoatCounter's export API into a local SQLite file; running `quinto` prints the most recent visits as a plain table. No TUI yet — this is the tracer bullet that proves the whole path works end to end.

Running `sync` twice must not duplicate anything or refetch what it already has. GoatCounter makes this easy rather than inventing your own bookkeeping: the export response carries a `last_hit_id`, and a new export accepts `start_from_hit_id`. **Store their cursor; don't invent a watermark.**

The export is asynchronous — create it, poll until finished, download the gzipped result. Their docs note the JSON export is preferable to CSV for most uses; check it before committing to CSV parsing.

The local schema is the spine of everything downstream and is what an agent reads, so it should be legible to something that has never seen the codebase. GoatCounter's export is flat — one row per hit, with `session` grouping them — so keep it flat and derive sessions as a view:

```sql
hits(hit_id PK, session, path, title, is_event,
     browser, system, bot, referrer, referrer_scheme,
     screen_size, country, first_visit, created_at)

sessions  -- a VIEW over hits, grouped by session:
          -- first_seen, last_seen, page_count, entry_path,
          -- referrer, country, browser
```

Keep `bot` in the table rather than filtering at sync time. The exclusion should be a `WHERE` clause the user can lift, not silent data loss at ingest.

**Their export format is versioned** — the version number is the first field of the header, and their docs strongly recommend erroring out on an unexpected version rather than mis-parsing into a corrupt local database. Do that.

This is the first ticket in an empty repo, so it also establishes the project skeleton: module setup, dependency choices, config loading, and the testing convention the rest of the tickets inherit.

**Testing policy for this project:** test the sync logic, not the rendering. Cursor handling, idempotency and version checking are claims you cannot verify by eyeballing a terminal — they need fixtures and assertions. TUI output does not get tests; it gets looked at. This is a two-weekend portfolio project, and that split is where tests actually pay for themselves.

**Blocked by:** A GoatCounter account with **"Individual pageviews" enabled** in site settings (off by default — without it the export contains nothing) and an API key. A human prerequisite, tracked in `PLAN.md` › Offen — bei Till. No other tickets.

**Status:** ready-for-agent

- [ ] `quinto sync` authenticates with a GoatCounter API key, creates an export, waits for it to finish, downloads and ingests it
- [ ] Individual pageviews land in a local SQLite file, one row per hit
- [ ] A `sessions` view groups hits into visits
- [ ] Running `quinto sync` twice in a row produces no duplicate rows
- [ ] Sync is incremental — subsequent runs pass the stored `last_hit_id` as `start_from_hit_id`
- [ ] An unexpected export format version aborts with a clear error instead of writing corrupt data
- [ ] Cursor handling, idempotency and version checking are covered by tests against fixture exports
- [ ] `quinto` prints the most recent visits as a readable table, excluding bots by default
- [ ] A visit you make to your own site appears after a sync
- [ ] The database file lives in a conventional per-user location, and its path is discoverable from the CLI
- [ ] Credentials are read from config or environment, never committed
