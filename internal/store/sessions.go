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

	// Match is the page inside this visit that a filter matched on, empty when
	// the match is already visible on the row or nothing is filtered. It
	// describes the query rather than the visit, which is why it is a plain
	// string and not part of the schema an agent reads.
	Match string
}

// SessionHit is one step within a visit.
type SessionHit struct {
	Path      string
	Title     string
	IsEvent   bool
	CreatedAt time.Time
}

// RecentSessions returns the most recent visits matching the filter, newest
// first. A zero SessionFilter returns recent visits with bots excluded.
func (db *DB) RecentSessions(ctx context.Context, limit int, f SessionFilter) ([]Session, error) {
	query, args := f.SQL(limit)

	rows, err := db.QueryContext(ctx, query, args...)
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
			&s.Duration, &s.Match); err != nil {
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
