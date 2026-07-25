# 01 — Sync one real visit from GoatCounter to the terminal

**What to build:** You visit your own site, run two commands, and see that visit in your terminal. `quinto sync` pulls individual pageviews from GoatCounter's export API into a local SQLite file; running `quinto` prints the most recent visits as a plain table. No TUI yet — this is the tracer bullet that proves the whole path works end to end.

Running `sync` twice must not duplicate anything or refetch what it already has. GoatCounter makes this easy rather than inventing your own bookkeeping: the export response carries a `last_hit_id`, and a new export accepts `start_from_hit_id`. **Store their cursor; don't invent a watermark.**

The export is asynchronous — create it, poll until finished, download the result.

**Verified against the live API 2026-07-25, and it differs from the docs in three ways:**

1. **`format` must be sent explicitly.** The spec says it defaults to CSV; posting `{}` returns `unknown format: ""`. Send `{"format":"csv"}`.
2. **Use CSV, not JSON** — despite GoatCounter's docs recommending JSON. Their OpenAPI spec is explicit that `start_from_hit_id` is the *CSV* cursor; JSON only offers `start_from_day`. Hit-level resumption beats day-level.
3. **Exports are rate limited to about one per hour** — a separate, much tighter budget than the documented 4 req/s. This is the single most important constraint on this ticket.
4. **The API returns transient `404 not found` on valid authenticated requests.** Observed twice on `/api/v0/me`, with five consecutive 200s immediately afterwards using identical credentials. A 404 is therefore not proof of a wrong URL or a bad token — retry idempotent GETs before surfacing an error, or the tool will report failures that aren't real.

Observed response shapes: `POST /export` returns `202` with `{id, site_id, format, last_hit_id, path}`; poll until `finished_at` is non-null; download returns `content-type: application/gzip` and must be gunzipped. An export with no data returns a 23-byte gzip containing **zero lines — not even a header row.** Handle that as "no data", not as a parse failure.

**The rate limit changes what sync is.** It is a deliberate, infrequent action. It cannot be run on demand, and it cannot be run twice to verify something. So: a `429` is a normal operating state, not a failure — capture the retry window from the response and tell the user when the next sync is possible. And the tool must be able to say **how old its data is**, because at an hourly cadence, silently stale numbers are the interface lying.

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

**Token scope — least privilege.** GoatCounter tokens carry granular permissions. quinto calls three endpoints, all of them export: create, poll, download. It therefore needs **Export and nothing else**, scoped to a single site rather than "all sites, including those created in the future".

It does not need *Read statistics* — every aggregate is computed locally, which is the whole point of the architecture. It must never hold *Record pageviews* (that writes events into the user's analytics) or *Create/Update sites* (that mutates their account). A read-only export token that leaks costs its owner a copy of data they already own; a token with write scope costs them their data's integrity.

This matters beyond one account: whatever scope we use becomes the documented default in the README, so every future user inherits it.

**Blocked by:** A GoatCounter account with **"Individual pageviews" enabled** in site settings (off by default — without it the export contains nothing) and an **Export-scoped API token**. A human prerequisite, tracked in `PLAN.md` › Offen — bei Till. No other tickets.

**Status:** resolved (2026-07-25)

- [x] `quinto sync` authenticates with a GoatCounter API key, creates an export, waits for it to finish, downloads and ingests it
- [x] Individual pageviews land in a local SQLite file, one row per hit
- [x] A `sessions` view groups hits into visits
- [x] Ingesting the same export twice produces no duplicate rows (verified against fixtures — a second live sync is impossible within the rate limit)
- [x] Sync is incremental — subsequent runs pass the stored `last_hit_id` as `start_from_hit_id`
- [x] A `429` is reported as "next sync available in Xm", not as an error or a stack trace
- [x] The stored data's age is recorded and displayable, so stale numbers are never shown as current
- [x] An unexpected export format version aborts with a clear error instead of writing corrupt data
- [x] Cursor handling, idempotency, version checking and 429 handling are covered by tests against fixture exports
- [x] `quinto` prints the most recent visits as a readable table, excluding bots by default
- [x] A visit you make to your own site appears after a sync
- [x] The database file lives in a conventional per-user location, and its path is discoverable from the CLI
- [x] Credentials are read from config or environment, never committed

## Answer

Closed 2026-07-25. Verified end to end against the live GoatCounter API, not
only against fixtures.

```
$ quinto sync
first sync — fetching everything GoatCounter has…
Synced 8 new pageviews.

$ quinto
data as of 21:04 (0s ago) · 8 pageviews stored

time   country  browser      referrer     pages  duration  entry
12:26  DE       Safari 26.5  direct       5      5s        /
11:32  DE       Chrome 126   Hacker News  3      8s        /quinto-test

$ quinto sync          # second run
syncing from hit 1657819971…
Nothing to do — GoatCounter allows about one export an hour.
Next sync possible in 1h0m0s.
```

Counts unchanged after the second sync (8 hits, 2 sessions): the stored cursor
advanced and a rate-limited run left state untouched.

**What the real export changed.** Four documented details were wrong, and each
would have produced a parser that passed its own tests and failed on contact:
the header is `2Path` (version glued to the column name, one field, not the
`2,Path` the docs render); the column is `UserAgent`, not `User-Agent`;
`Screen size` is a quoted comma-bearing field, so splitting on `,` shifts every
later column; and rows carry **no id at all**, which made the planned
`hit_id` primary key impossible. The key is now a content hash — which is also
what makes re-ingest a no-op.

It also disproved a design assumption: `first_visit` is set on more than one
hit per session, so "the flagged hit" is not unique. The sessions view now
takes the *earliest* flagged hit, preserving the decision's intent while being
deterministic.

**Left open, deliberately:** Till's own pageviews were initially absent, which
looked like an ad-blocker problem. Real data shows his traffic arriving
normally (the Safari session above). Not a quinto issue.
