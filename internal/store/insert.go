package store

import (
	"context"
	"fmt"
	"time"
)

// Hit is one pageview as stored locally. It mirrors the export's columns; the
// goatcounter package produces these and store consumes them, so neither
// depends on the other's types.
type Hit struct {
	Key            string
	Path           string
	Title          string
	IsEvent        bool
	Browser        string
	System         string
	Session        string
	Bot            int
	Referrer       string
	ReferrerScheme string
	ScreenSize     string
	Country        string
	FirstVisit     bool
	CreatedAt      string
}

// InsertHits writes hits and reports how many were new. Existing keys are
// ignored rather than replaced, which makes re-ingesting the same export a
// no-op — the property that lets sync be run without fear.
func (db *DB) InsertHits(ctx context.Context, hits []Hit) (inserted int, err error) {
	if len(hits) == 0 {
		return 0, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO hits
			(hit_key, session, path, title, is_event, browser, system, bot,
			 referrer, referrer_scheme, screen_size, country, first_visit, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("preparing insert: %w", err)
	}
	defer stmt.Close()

	for _, h := range hits {
		res, err := stmt.ExecContext(ctx,
			h.Key, h.Session, h.Path, h.Title, boolToInt(h.IsEvent), h.Browser,
			h.System, h.Bot, h.Referrer, h.ReferrerScheme, h.ScreenSize,
			h.Country, boolToInt(h.FirstVisit), h.CreatedAt)
		if err != nil {
			return 0, fmt.Errorf("inserting hit %s: %w", h.Key, err)
		}
		if n, err := res.RowsAffected(); err == nil {
			inserted += int(n)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing hits: %w", err)
	}
	return inserted, nil
}

// SyncState describes the last sync, so the interface can say how old its
// numbers are instead of presenting stale data as current.
type SyncState struct {
	LastHitID    int64
	LastSyncedAt time.Time
	SyncedRows   int64
	Synced       bool
}

// SyncState reads the recorded cursor and timing.
func (db *DB) SyncState(ctx context.Context) (SyncState, error) {
	var (
		s     SyncState
		id    *int64
		at    *string
		rows  int64
		found bool
	)

	err := db.QueryRowContext(ctx,
		`SELECT last_hit_id, last_synced_at, synced_rows FROM sync_state WHERE id = 1`).
		Scan(&id, &at, &rows)
	switch {
	case err != nil && err.Error() == "sql: no rows in result set":
		return s, nil
	case err != nil:
		return s, fmt.Errorf("reading sync state: %w", err)
	}
	found = true

	if id != nil {
		s.LastHitID = *id
	}
	if at != nil {
		if t, perr := time.Parse(time.RFC3339, *at); perr == nil {
			s.LastSyncedAt = t
		}
	}
	s.SyncedRows = rows
	s.Synced = found
	return s, nil
}

// RecordSync stores GoatCounter's cursor and when we last spoke to them.
//
// The cursor only ever moves forward. An empty incremental export currently
// echoes the cursor back unchanged, but nothing in their API guarantees that —
// and a response carrying 0 would otherwise reset us to the beginning and
// refetch everything on the next run. Taking the maximum makes that class of
// failure impossible rather than unlikely.
func (db *DB) RecordSync(ctx context.Context, lastHitID int64, at time.Time, rows int64) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO sync_state (id, last_hit_id, last_synced_at, synced_rows)
		VALUES (1, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			last_hit_id    = MAX(COALESCE(sync_state.last_hit_id, 0), excluded.last_hit_id),
			last_synced_at = excluded.last_synced_at,
			synced_rows    = sync_state.synced_rows + excluded.synced_rows`,
		lastHitID, at.UTC().Format(time.RFC3339), rows)
	if err != nil {
		return fmt.Errorf("recording sync: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
