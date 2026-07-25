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

// sessionsDDL turns hits into visits. This is the view the stream view reads,
// so its shape decides what every row on screen can show.
//
// TODO(till): define this view. See the note in the package docs of
// schema_test.go for what the stream view needs from it.
const sessionsDDL = ``

// allDDL runs in order on every open. Statements are idempotent so this
// doubles as the migration path until there's a reason for something heavier.
var allDDL = []string{hitsDDL, syncStateDDL, sessionsDDL}
