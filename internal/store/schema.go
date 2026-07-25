// Package store owns quinto's local SQLite file — the one both the TUI and
// `quinto query` read. Nothing in here talks to the network.
//
// The schema stays flat and close to GoatCounter's export: one row per hit,
// with `session` grouping hits into a visit. It is also quinto's agent
// interface, so column names favour legibility over brevity. An agent that has
// never seen this codebase should be able to write a correct query from the
// schema alone.
package store

// hitsDDL mirrors GoatCounter's individual-pageview export, one column at a
// time, so there is no translation layer to reason about.
//
// The primary key is a content hash, not GoatCounter's id: their CSV export
// carries no row identifier at all. The export job reports an overall
// last_hit_id for use as a sync cursor, but individual rows are anonymous.
// Hashing the row makes re-ingesting the same export a no-op, and the only
// rows that can collide are identical in every field — indistinguishable in
// any case.
//
// Bots are stored, never filtered at ingest. Excluding them is a WHERE clause
// the user can lift; dropping them is data we can't get back.
const hitsDDL = `
CREATE TABLE IF NOT EXISTS hits (
	hit_key         TEXT PRIMARY KEY,             -- sha256 of the export row
	session         TEXT    NOT NULL,             -- salted hash, rotates daily by design
	path            TEXT    NOT NULL,             -- also the event name when is_event = 1
	title           TEXT,
	is_event        INTEGER NOT NULL DEFAULT 0,
	browser         TEXT,
	system          TEXT,
	bot             INTEGER NOT NULL DEFAULT 0,   -- 0 = human, non-zero = bot classification
	referrer        TEXT,                         -- normalised by GoatCounter, e.g. "Hacker News"
	referrer_scheme TEXT,                         -- h link, g generated, c campaign, o other
	screen_size     TEXT,                         -- "width,height,scale"
	country         TEXT,                         -- ISO 3166-2
	first_visit     INTEGER NOT NULL DEFAULT 0,   -- 1 = GoatCounter marked this a visit's first hit
	created_at      TEXT    NOT NULL              -- RFC 3339
);

CREATE INDEX IF NOT EXISTS hits_created_at ON hits (created_at DESC);
CREATE INDEX IF NOT EXISTS hits_session    ON hits (session);
`

// syncStateDDL holds GoatCounter's own cursor rather than a timestamp we
// invented. Their export returns last_hit_id and accepts start_from_hit_id, so
// incremental sync is their feature, not our bookkeeping. synced_rows and
// last_synced_at exist so the interface can say how old its data is — at a
// one-export-per-hour rate limit, silently stale numbers would be a lie.
const syncStateDDL = `
CREATE TABLE IF NOT EXISTS sync_state (
	id             INTEGER PRIMARY KEY CHECK (id = 1),
	last_hit_id    INTEGER,
	last_synced_at TEXT,
	synced_rows    INTEGER NOT NULL DEFAULT 0
);
`

// sessionsDDL turns hits into visits. The stream view reads this, so its shape
// decides what every row on screen can show, and it's the first thing an agent
// will reach for.
//
// Four decisions are baked in (Till, 2026-07-25):
//
//  1. Midnight split: accepted, not stitched. GoatCounter's session hash
//     rotates daily, so a visit crossing midnight appears as two sessions.
//     Stitching would mean guessing that two hashes are the same person, which
//     is exactly the inference their design refuses to make.
//  2. Entry hit: trust GoatCounter's first_visit flag. Real exports turned out
//     to set that flag on more than one hit in a session, so "the flagged hit"
//     is not unique — the entry is therefore the EARLIEST flagged hit, which
//     keeps the intent and makes the result deterministic. When no hit in the
//     session is flagged, entry fields stay NULL rather than promoting some
//     later hit to "the entry".
//  3. Single-page visits: duration is NULL, not 0. You cannot observe when
//     someone left, so 0 would assert something untrue. NULL also keeps
//     aggregates sane — AVG skips it, so "average visit duration" means the
//     average of visits we could actually measure, instead of being dragged
//     toward zero by every unmeasurable one. At this traffic most visits are
//     single-page, so the difference is not cosmetic.
//  4. Bots stay in. Filtering here would make exclusion invisible and
//     permanent; leaving them keeps it a WHERE clause the user can lift.
//
// Recreated on every open so the definition always matches this file.
const sessionsDDL = `
DROP VIEW IF EXISTS sessions;
CREATE VIEW sessions AS
WITH ranked AS (
	SELECT session, path, referrer, referrer_scheme, country, browser, system,
	       first_visit,
	       ROW_NUMBER() OVER (
	           PARTITION BY session
	           ORDER BY first_visit DESC, created_at ASC, hit_key ASC
	       ) AS rn
	FROM hits
),
agg AS (
	SELECT session,
	       MIN(created_at) AS first_seen,
	       MAX(created_at) AS last_seen,
	       COUNT(*)        AS page_count,
	       MAX(bot)        AS bot,
	       -- NULL for single-page visits: unmeasurable, not zero.
	       CASE WHEN COUNT(*) > 1
	            THEN unixepoch(MAX(created_at)) - unixepoch(MIN(created_at))
	       END AS duration_seconds
	FROM hits
	GROUP BY session
)
SELECT a.session,
       a.first_seen,
       a.last_seen,
       a.page_count,
       a.bot,
       CASE WHEN r.first_visit = 1 THEN r.path            END AS entry_path,
       CASE WHEN r.first_visit = 1 THEN r.referrer        END AS referrer,
       CASE WHEN r.first_visit = 1 THEN r.referrer_scheme END AS referrer_scheme,
       CASE WHEN r.first_visit = 1 THEN r.country         END AS country,
       CASE WHEN r.first_visit = 1 THEN r.browser         END AS browser,
       CASE WHEN r.first_visit = 1 THEN r.system          END AS system,
       a.duration_seconds
FROM agg a
JOIN ranked r ON r.session = a.session AND r.rn = 1;
`

// allDDL runs in order on every open. Statements are idempotent so this
// doubles as the migration path until there's a reason for something heavier.
var allDDL = []string{hitsDDL, syncStateDDL, sessionsDDL}
