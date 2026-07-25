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
// Bots are stored, never filtered at ingest. Excluding them is a WHERE clause
// the user can lift; dropping them is data we can't get back.
const hitsDDL = `
CREATE TABLE IF NOT EXISTS hits (
	hit_id          INTEGER PRIMARY KEY,          -- GoatCounter's id; doubles as the sync cursor
	session         TEXT    NOT NULL,             -- salted hash, rotates daily by design
	path            TEXT    NOT NULL,             -- also the event name when is_event = 1
	title           TEXT,
	is_event        INTEGER NOT NULL DEFAULT 0,
	browser         TEXT,
	system          TEXT,
	bot             INTEGER NOT NULL DEFAULT 0,   -- 0 = human, non-zero = bot classification
	referrer        TEXT,
	referrer_scheme TEXT,                         -- h link, g generated, c campaign, o other
	screen_size     TEXT,
	country         TEXT,                         -- ISO 3166-2
	first_visit     INTEGER NOT NULL DEFAULT 0,   -- 1 = first hit of this session
	created_at      TEXT    NOT NULL              -- RFC 3339
);

CREATE INDEX IF NOT EXISTS hits_created_at ON hits (created_at DESC);
CREATE INDEX IF NOT EXISTS hits_session    ON hits (session);
`

// syncStateDDL holds GoatCounter's own cursor rather than a timestamp we
// invented. Their export returns last_hit_id and accepts start_from_hit_id, so
// incremental sync is their feature, not our bookkeeping.
const syncStateDDL = `
CREATE TABLE IF NOT EXISTS sync_state (
	id            INTEGER PRIMARY KEY CHECK (id = 1),  -- single row
	last_hit_id   INTEGER,
	last_synced_at TEXT
);
`

// sessionsDDL turns hits into visits. The stream view reads this, so its shape
// decides what every row on screen can show, and it's the first thing an agent
// will reach for. Getting it right matters more than anything else in this file.
//
// Target shape — one row per visit:
//
//	▼ 14:22  DE · Chrome · google.com     3 pages · 2m14s
//
// So at minimum: session, first_seen, last_seen, page_count, entry_path,
// referrer, country, browser.
//
// Four decisions are baked in (Till, 2026-07-25):
//
//  1. Midnight split: accepted, not stitched. GoatCounter's session hash
//     rotates daily, so a visit crossing midnight appears as two sessions.
//     Stitching would mean guessing that two hashes are the same person, which
//     is exactly the inference their design refuses to make.
//  2. Entry hit: trust GoatCounter's first_visit flag rather than deriving it
//     from the earliest timestamp. Consequence — if a session's entry hit
//     predates the first sync, entry_path and referrer are NULL rather than
//     wrong. Honest, and self-correcting once the entry hit is synced.
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
SELECT
	session,
	MIN(created_at) AS first_seen,
	MAX(created_at) AS last_seen,
	COUNT(*)        AS page_count,
	MAX(bot)        AS bot,

	MAX(CASE WHEN first_visit = 1 THEN path            END) AS entry_path,
	MAX(CASE WHEN first_visit = 1 THEN referrer        END) AS referrer,
	MAX(CASE WHEN first_visit = 1 THEN referrer_scheme END) AS referrer_scheme,
	MAX(CASE WHEN first_visit = 1 THEN country         END) AS country,
	MAX(CASE WHEN first_visit = 1 THEN browser         END) AS browser,
	MAX(CASE WHEN first_visit = 1 THEN system          END) AS system,

	-- NULL for single-page visits: unmeasurable, not zero.
	CASE WHEN COUNT(*) > 1
		THEN unixepoch(MAX(created_at)) - unixepoch(MIN(created_at))
	END AS duration_seconds
FROM hits
GROUP BY session;
`

// allDDL runs in order on every open. Statements are idempotent so this
// doubles as the migration path until there's a reason for something heavier.
var allDDL = []string{hitsDDL, syncStateDDL, sessionsDDL}
