package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Session is one visit, as the stream view renders it. Nullable fields are
// genuinely nullable: entry details are absent when no hit in the session was
// flagged as the entry, and duration is absent when the visit produced only
// one observation. Both are "unknown", not "empty", and the UI must be able to
// tell the difference.
type Session struct {
	ID         string
	FirstSeen  time.Time
	LastSeen   time.Time
	PageCount  int
	EventCount int
	Bot        int

	EntryPath sql.NullString
	Referrer  sql.NullString
	Country   sql.NullString
	Browser   sql.NullString
	System    sql.NullString
	Duration  sql.NullInt64
}

// SessionHit is one step within a visit.
type SessionHit struct {
	Path      string
	Title     string
	IsEvent   bool
	CreatedAt time.Time
}

// RecentSessions returns the most recent visits, newest first.
func (db *DB) RecentSessions(ctx context.Context, limit int, includeBots bool) ([]Session, error) {
	where := "WHERE bot = 0"
	if includeBots {
		where = ""
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT session, first_seen, last_seen, page_count, event_count, bot,
		       entry_path, referrer, country, browser, system, duration_seconds
		FROM sessions %s
		ORDER BY first_seen DESC
		LIMIT ?`, where), limit)
	if err != nil {
		return nil, fmt.Errorf("loading sessions: %w", err)
	}
	defer rows.Close()

	var out []Session
	for rows.Next() {
		var (
			s                   Session
			firstSeen, lastSeen string
		)
		if err := rows.Scan(&s.ID, &firstSeen, &lastSeen, &s.PageCount, &s.EventCount,
			&s.Bot, &s.EntryPath, &s.Referrer, &s.Country, &s.Browser, &s.System,
			&s.Duration); err != nil {
			return nil, fmt.Errorf("scanning session: %w", err)
		}
		s.FirstSeen, _ = time.Parse(time.RFC3339, firstSeen)
		s.LastSeen, _ = time.Parse(time.RFC3339, lastSeen)
		out = append(out, s)
	}
	return out, rows.Err()
}

// SessionHits returns a visit's steps in the order they happened — the path
// through the site that the stream view expands to show.
func (db *DB) SessionHits(ctx context.Context, sessionID string) ([]SessionHit, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT path, COALESCE(title, ''), is_event, created_at
		FROM hits WHERE session = ?
		ORDER BY created_at ASC, hit_key ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("loading session hits: %w", err)
	}
	defer rows.Close()

	var out []SessionHit
	for rows.Next() {
		var (
			h         SessionHit
			isEvent   int
			createdAt string
		)
		if err := rows.Scan(&h.Path, &h.Title, &isEvent, &createdAt); err != nil {
			return nil, fmt.Errorf("scanning hit: %w", err)
		}
		h.IsEvent = isEvent == 1
		h.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		out = append(out, h)
	}
	return out, rows.Err()
}

// Totals summarises what is stored, for the header line.
type Totals struct {
	Sessions int
	Hits     int
	Bots     int
}

func (db *DB) Totals(ctx context.Context) (Totals, error) {
	var t Totals
	err := db.QueryRowContext(ctx, `
		SELECT (SELECT count(*) FROM sessions WHERE bot = 0),
		       (SELECT count(*) FROM hits     WHERE bot = 0),
		       (SELECT count(*) FROM sessions WHERE bot > 0)`).
		Scan(&t.Sessions, &t.Hits, &t.Bots)
	if err != nil {
		return t, fmt.Errorf("reading totals: %w", err)
	}
	return t, nil
}
