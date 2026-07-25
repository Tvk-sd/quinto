package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the local SQLite file.
type DB struct {
	*sql.DB
	Path string
}

// DefaultPath returns the conventional per-user location for the database.
// Honours XDG_DATA_HOME so it behaves on Linux as well as macOS.
func DefaultPath(name string) (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locating home directory: %w", err)
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "quinto", name), nil
}

// Open opens (creating if needed) the database at path and applies the schema.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	db := &DB{DB: sqlDB, Path: path}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// migrate applies every DDL statement. All of them are idempotent, so this is
// safe to run on every open.
func (db *DB) migrate() error {
	for _, ddl := range allDDL {
		if ddl == "" {
			continue // not yet defined
		}
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("applying schema: %w", err)
		}
	}
	return nil
}

// LastHitID returns GoatCounter's cursor from the previous sync, and whether
// one has been recorded. A missing cursor means "sync everything".
func (db *DB) LastHitID() (int64, bool, error) {
	var id sql.NullInt64
	err := db.QueryRow(`SELECT last_hit_id FROM sync_state WHERE id = 1`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("reading sync cursor: %w", err)
	}
	return id.Int64, id.Valid, nil
}

// SetLastHitID records the cursor returned by GoatCounter's export.
func (db *DB) SetLastHitID(id int64, syncedAt string) error {
	_, err := db.Exec(`
		INSERT INTO sync_state (id, last_hit_id, last_synced_at) VALUES (1, ?, ?)
		ON CONFLICT (id) DO UPDATE SET last_hit_id = excluded.last_hit_id,
		                               last_synced_at = excluded.last_synced_at`,
		id, syncedAt)
	if err != nil {
		return fmt.Errorf("writing sync cursor: %w", err)
	}
	return nil
}
