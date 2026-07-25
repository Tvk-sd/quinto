package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// openTemp gives each test its own database file.
func openTemp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertHit(t *testing.T, db *DB, id int64, session, path, createdAt string, firstVisit, bot int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO hits (hit_id, session, path, created_at, first_visit, bot,
		                  referrer, referrer_scheme, country, browser, system)
		VALUES (?, ?, ?, ?, ?, ?, 'google.com', 'h', 'DE', 'Chrome', 'macOS')`,
		id, session, path, createdAt, firstVisit, bot)
	if err != nil {
		t.Fatalf("insert hit %d: %v", id, err)
	}
}

// The schema is applied by Open; if multi-statement DDL didn't execute, the
// indexes and view below would be missing.
func TestMigrateCreatesTablesAndView(t *testing.T) {
	db := openTemp(t)

	for _, name := range []string{"hits", "sync_state", "sessions"} {
		var got string
		err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE name = ?`, name).Scan(&got)
		if err != nil {
			t.Errorf("expected %q to exist: %v", name, err)
		}
	}

	// Open twice: DDL must be idempotent, since migrate runs on every open.
	db2, err := Open(db.Path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	db2.Close()
}

// Decision 3: a visit we cannot measure reports NULL, never 0.
func TestSinglePageVisitHasNullDuration(t *testing.T) {
	db := openTemp(t)
	insertHit(t, db, 1, "sess-solo", "/", "2026-07-25T14:19:10Z", 1, 0)

	var duration sql.NullInt64
	var pageCount int
	err := db.QueryRow(`
		SELECT duration_seconds, page_count FROM sessions WHERE session = 'sess-solo'`).
		Scan(&duration, &pageCount)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}

	if pageCount != 1 {
		t.Errorf("page_count = %d, want 1", pageCount)
	}
	if duration.Valid {
		t.Errorf("duration_seconds = %d, want NULL — a single-page visit's duration is unobservable", duration.Int64)
	}
}

func TestMultiPageVisitAggregates(t *testing.T) {
	db := openTemp(t)
	insertHit(t, db, 1, "sess-a", "/", "2026-07-25T14:22:00Z", 1, 0)
	insertHit(t, db, 2, "sess-a", "/pricing", "2026-07-25T14:23:00Z", 0, 0)
	insertHit(t, db, 3, "sess-a", "/signup", "2026-07-25T14:24:14Z", 0, 0)

	var (
		pageCount int
		duration  sql.NullInt64
		entryPath sql.NullString
		firstSeen string
		lastSeen  string
	)
	err := db.QueryRow(`
		SELECT page_count, duration_seconds, entry_path, first_seen, last_seen
		FROM sessions WHERE session = 'sess-a'`).
		Scan(&pageCount, &duration, &entryPath, &firstSeen, &lastSeen)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}

	if pageCount != 3 {
		t.Errorf("page_count = %d, want 3", pageCount)
	}
	if !duration.Valid || duration.Int64 != 134 {
		t.Errorf("duration_seconds = %v, want 134", duration)
	}
	// Decision 2: the entry comes from the first_visit flag, not MIN(created_at).
	if entryPath.String != "/" {
		t.Errorf("entry_path = %q, want %q", entryPath.String, "/")
	}
	if firstSeen != "2026-07-25T14:22:00Z" || lastSeen != "2026-07-25T14:24:14Z" {
		t.Errorf("bounds = %s..%s", firstSeen, lastSeen)
	}
}

// Decision 2's consequence: an entry hit we never synced yields NULL rather
// than silently promoting some later hit to "the entry".
func TestSessionWithoutEntryHitHasNullEntry(t *testing.T) {
	db := openTemp(t)
	insertHit(t, db, 10, "sess-partial", "/pricing", "2026-07-25T09:00:00Z", 0, 0)
	insertHit(t, db, 11, "sess-partial", "/signup", "2026-07-25T09:01:00Z", 0, 0)

	var entryPath sql.NullString
	err := db.QueryRow(
		`SELECT entry_path FROM sessions WHERE session = 'sess-partial'`).Scan(&entryPath)
	if err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	if entryPath.Valid {
		t.Errorf("entry_path = %q, want NULL when no hit is flagged first_visit", entryPath.String)
	}
}

// Decision 4: bots reach the view. Excluding them is the caller's WHERE clause.
func TestBotsAreVisibleInSessions(t *testing.T) {
	db := openTemp(t)
	insertHit(t, db, 20, "sess-bot", "/", "2026-07-25T04:00:00Z", 1, 1)
	insertHit(t, db, 21, "sess-human", "/", "2026-07-25T14:00:00Z", 1, 0)

	var total, humans int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&total); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE bot = 0`).Scan(&humans); err != nil {
		t.Fatalf("count human sessions: %v", err)
	}

	if total != 2 {
		t.Errorf("total sessions = %d, want 2 — bots must not be filtered by the view", total)
	}
	if humans != 1 {
		t.Errorf("human sessions = %d, want 1", humans)
	}
}

func TestSyncCursorRoundTrips(t *testing.T) {
	db := openTemp(t)

	if _, ok, err := db.LastHitID(); err != nil || ok {
		t.Fatalf("fresh database: got ok=%v err=%v, want no cursor", ok, err)
	}

	if err := db.SetLastHitID(42, "2026-07-25T12:00:00Z"); err != nil {
		t.Fatalf("SetLastHitID: %v", err)
	}
	id, ok, err := db.LastHitID()
	if err != nil || !ok || id != 42 {
		t.Fatalf("after set: id=%d ok=%v err=%v, want 42/true/nil", id, ok, err)
	}

	// Overwrites rather than accumulating rows.
	if err := db.SetLastHitID(99, "2026-07-25T13:00:00Z"); err != nil {
		t.Fatalf("SetLastHitID second: %v", err)
	}
	id, _, _ = db.LastHitID()
	if id != 99 {
		t.Errorf("cursor = %d, want 99", id)
	}
}
