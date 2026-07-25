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

// Cursor state lives in insert.go alongside the write path.
