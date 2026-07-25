package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// seeded returns a populated database path, closed and ready to reopen.
func seeded(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "q.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	insertHit(t, db, 1, "sess-a", "/", "2026-07-25T14:22:00Z", 1, 0)
	insertHit(t, db, 2, "sess-a", "/process", "2026-07-25T14:23:00Z", 0, 0)
	insertHit(t, db, 3, "sess-bot", "/", "2026-07-25T04:00:00Z", 1, 1)
	db.Close()
	return path
}

func TestQueryReturnsColumnsAndRows(t *testing.T) {
	db, err := OpenReadOnly(seeded(t))
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer db.Close()

	res, err := db.Query(context.Background(),
		`SELECT path, bot FROM hits ORDER BY hit_id`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if got := strings.Join(res.Columns, ","); got != "path,bot" {
		t.Errorf("columns = %q", got)
	}
	if len(res.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(res.Rows))
	}
	// Text must arrive as string, not []byte — JSON and tables both need it.
	if _, ok := res.Rows[0][0].(string); !ok {
		t.Errorf("path is %T, want string", res.Rows[0][0])
	}
}

// The query surface is what an agent drives, so it must not be able to change
// anything even if it tries.
func TestQueryIsReadOnly(t *testing.T) {
	db, err := OpenReadOnly(seeded(t))
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer db.Close()

	for _, stmt := range []string{
		`DELETE FROM hits`,
		`DROP TABLE hits`,
		`INSERT INTO hits (hit_id, session, path, created_at) VALUES (99, 's', '/x', '2026-01-01T00:00:00Z')`,
		`UPDATE hits SET path = '/hacked'`,
	} {
		if _, err := db.Query(context.Background(), stmt); err == nil {
			t.Errorf("%q was allowed — the query connection must be read-only", stmt)
		}
	}

	// And the data survived.
	res, err := db.Query(context.Background(), `SELECT count(*) FROM hits`)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n, _ := res.Rows[0][0].(int64); n != 3 {
		t.Errorf("hits = %d, want 3 — data was modified", n)
	}
}

// NULL must survive as nil so callers can distinguish "unknown" from "empty".
// duration_seconds on a single-page visit is the case that matters.
func TestNullSurvivesAsNil(t *testing.T) {
	db, err := OpenReadOnly(seeded(t))
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer db.Close()

	res, err := db.Query(context.Background(),
		`SELECT duration_seconds FROM sessions WHERE session = 'sess-bot'`)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.Rows[0][0] != nil {
		t.Errorf("duration = %v, want nil for a single-page visit", res.Rows[0][0])
	}
}

func TestSchemaIsDiscoverable(t *testing.T) {
	db, err := OpenReadOnly(seeded(t))
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer db.Close()

	ddl, err := db.Schema(context.Background())
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}

	// An agent that has never seen this project must be able to write a
	// correct query from this output alone.
	for _, want := range []string{
		"CREATE TABLE", "hits", "CREATE VIEW", "sessions",
		"bot", "duration_seconds", "first_visit", "referrer",
	} {
		if !strings.Contains(ddl, want) {
			t.Errorf("schema output is missing %q", want)
		}
	}
}

func TestBadSQLReturnsUsableError(t *testing.T) {
	db, err := OpenReadOnly(seeded(t))
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer db.Close()

	_, err = db.Query(context.Background(), `SELECT pth FROM hits`)
	if err == nil {
		t.Fatal("expected an error for an unknown column")
	}
	if !strings.Contains(err.Error(), "pth") {
		t.Errorf("error %q should name the offending column", err)
	}
}

func TestOpenReadOnlyMissingFileExplainsItself(t *testing.T) {
	_, err := OpenReadOnly(filepath.Join(t.TempDir(), "absent.db"))
	if err == nil {
		t.Fatal("expected an error")
	}
	// The message should tell a first-time user what to do next.
	if !strings.Contains(err.Error(), "sync") {
		t.Errorf("error %q should point at `quinto sync`", err)
	}
}
