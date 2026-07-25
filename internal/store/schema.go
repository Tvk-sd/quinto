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
// TODO(till): write this view. Four decisions are baked into it:
//
//  1. Midnight split. GoatCounter's session hash rotates daily, so a visit
//     spanning midnight arrives as two sessions. Accept it and document the
//     quirk, or try to stitch across the boundary?
//  2. Entry hit. GoatCounter flags it with first_visit = 1. Trust that flag, or
//     derive it as the earliest created_at in the session? They can disagree.
//  3. One-page visits. last_seen - first_seen is 0, but the real duration is
//     unknown — you can't see when someone left. Report 0, or NULL?
//  4. Bots. Filter them here so callers can't forget, or leave them in so the
//     exclusion stays a WHERE clause the user can lift? PLAN.md argues for the
//     second; this view is where that argument gets tested.
const sessionsDDL = ``

// allDDL runs in order on every open. Statements are idempotent so this
// doubles as the migration path until there's a reason for something heavier.
var allDDL = []string{hitsDDL, syncStateDDL, sessionsDDL}
