package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
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

// hit builds a Hit with sensible defaults so tests only state what matters.
func hit(key, session, path, createdAt string, firstVisit bool, bot int) Hit {
	return Hit{
		Key: key, Session: session, Path: path, CreatedAt: createdAt,
		FirstVisit: firstVisit, Bot: bot,
		Referrer: "Hacker News", ReferrerScheme: "g",
		Country: "DE", Browser: "Chrome 126", System: "macOS 10.15",
	}
}

func insert(t *testing.T, db *DB, hits ...Hit) {
	t.Helper()
	if _, err := db.InsertHits(context.Background(), hits); err != nil {
		t.Fatalf("InsertHits: %v", err)
	}
}

func TestMigrateCreatesTablesAndView(t *testing.T) {
	db := openTemp(t)

	for _, name := range []string{"hits", "sync_state", "sessions"} {
		var got string
		if err := db.QueryRow(
			`SELECT name FROM sqlite_master WHERE name = ?`, name).Scan(&got); err != nil {
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
	insert(t, db, hit("k1", "sess-solo", "/", "2026-07-25T14:19:10Z", true, 0))

	var duration sql.NullInt64
	var pageCount int
	if err := db.QueryRow(
		`SELECT duration_seconds, page_count FROM sessions WHERE session = 'sess-solo'`).
		Scan(&duration, &pageCount); err != nil {
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
	insert(t, db,
		hit("k1", "sess-a", "/", "2026-07-25T14:22:00Z", true, 0),
		hit("k2", "sess-a", "/pricing", "2026-07-25T14:23:00Z", false, 0),
		hit("k3", "sess-a", "/signup", "2026-07-25T14:24:14Z", false, 0),
	)

	var (
		pageCount int
		duration  sql.NullInt64
		entryPath sql.NullString
		firstSeen string
		lastSeen  string
	)
	if err := db.QueryRow(`
		SELECT page_count, duration_seconds, entry_path, first_seen, last_seen
		FROM sessions WHERE session = 'sess-a'`).
		Scan(&pageCount, &duration, &entryPath, &firstSeen, &lastSeen); err != nil {
		t.Fatalf("query sessions: %v", err)
	}

	if pageCount != 3 {
		t.Errorf("page_count = %d, want 3", pageCount)
	}
	if !duration.Valid || duration.Int64 != 134 {
		t.Errorf("duration_seconds = %v, want 134", duration)
	}
	if entryPath.String != "/" {
		t.Errorf("entry_path = %q, want %q", entryPath.String, "/")
	}
	if firstSeen != "2026-07-25T14:22:00Z" || lastSeen != "2026-07-25T14:24:14Z" {
		t.Errorf("bounds = %s..%s", firstSeen, lastSeen)
	}
}

// Real exports flag first_visit on more than one hit in a session, so the
// entry must be the EARLIEST flagged hit — deterministically, not whichever
// row an aggregate happens to pick.
func TestEntryIsEarliestFlaggedHit(t *testing.T) {
	db := openTemp(t)
	insert(t, db,
		hit("k1", "sess-m", "/zzz-later", "2026-07-25T14:30:00Z", true, 0),
		hit("k2", "sess-m", "/aaa-entry", "2026-07-25T14:20:00Z", true, 0),
		hit("k3", "sess-m", "/middle", "2026-07-25T14:25:00Z", false, 0),
	)

	var entryPath string
	if err := db.QueryRow(
		`SELECT entry_path FROM sessions WHERE session = 'sess-m'`).Scan(&entryPath); err != nil {
		t.Fatalf("query: %v", err)
	}
	if entryPath != "/aaa-entry" {
		t.Errorf("entry_path = %q, want /aaa-entry (earliest flagged hit)", entryPath)
	}
}

// Decision 2's consequence: an entry hit we never synced yields NULL rather
// than silently promoting some later hit to "the entry".
func TestSessionWithoutEntryHitHasNullEntry(t *testing.T) {
	db := openTemp(t)
	insert(t, db,
		hit("k1", "sess-partial", "/pricing", "2026-07-25T09:00:00Z", false, 0),
		hit("k2", "sess-partial", "/signup", "2026-07-25T09:01:00Z", false, 0),
	)

	var entryPath sql.NullString
	if err := db.QueryRow(
		`SELECT entry_path FROM sessions WHERE session = 'sess-partial'`).Scan(&entryPath); err != nil {
		t.Fatalf("query sessions: %v", err)
	}
	if entryPath.Valid {
		t.Errorf("entry_path = %q, want NULL when no hit is flagged first_visit", entryPath.String)
	}
}

// Decision 4: bots reach the view. Excluding them is the caller's WHERE clause.
func TestBotsAreVisibleInSessions(t *testing.T) {
	db := openTemp(t)
	insert(t, db,
		hit("k1", "sess-bot", "/", "2026-07-25T04:00:00Z", true, 1),
		hit("k2", "sess-human", "/", "2026-07-25T14:00:00Z", true, 0),
	)

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

// Re-ingesting the same export must be a no-op. This is the property that
// makes sync safe to run, and it cannot be verified live: the export rate
// limit means a second real sync is impossible within the hour.
func TestReinsertingSameHitsIsANoOp(t *testing.T) {
	db := openTemp(t)
	hits := []Hit{
		hit("k1", "sess-a", "/", "2026-07-25T14:22:00Z", true, 0),
		hit("k2", "sess-a", "/pricing", "2026-07-25T14:23:00Z", false, 0),
	}

	first, err := db.InsertHits(context.Background(), hits)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	second, err := db.InsertHits(context.Background(), hits)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}

	if first != 2 {
		t.Errorf("first insert reported %d new rows, want 2", first)
	}
	if second != 0 {
		t.Errorf("second insert reported %d new rows, want 0", second)
	}

	var total int
	db.QueryRow(`SELECT COUNT(*) FROM hits`).Scan(&total)
	if total != 2 {
		t.Errorf("hits = %d, want 2 — re-ingest duplicated rows", total)
	}
}

func TestSyncStateRoundTrips(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	s, err := db.SyncState(ctx)
	if err != nil {
		t.Fatalf("SyncState: %v", err)
	}
	if s.Synced {
		t.Error("a fresh database must report no previous sync")
	}

	at := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	if err := db.RecordSync(ctx, 1657794647, at, 3); err != nil {
		t.Fatalf("RecordSync: %v", err)
	}

	s, err = db.SyncState(ctx)
	if err != nil {
		t.Fatalf("SyncState: %v", err)
	}
	if !s.Synced || s.LastHitID != 1657794647 || s.SyncedRows != 3 {
		t.Fatalf("got %+v", s)
	}
	if !s.LastSyncedAt.Equal(at) {
		t.Errorf("LastSyncedAt = %v, want %v", s.LastSyncedAt, at)
	}

	// A second sync advances the cursor and accumulates the row count.
	if err := db.RecordSync(ctx, 1657794700, at.Add(time.Hour), 5); err != nil {
		t.Fatalf("second RecordSync: %v", err)
	}
	s, _ = db.SyncState(ctx)
	if s.LastHitID != 1657794700 || s.SyncedRows != 8 {
		t.Errorf("after second sync: %+v, want cursor 1657794700 and 8 rows", s)
	}
}
