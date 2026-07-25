package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// Result is a query's output, kept as plain values so callers can render it
// as a table or as JSON without knowing anything about the schema.
type Result struct {
	Columns []string
	Rows    [][]any
}

// OpenReadOnly opens the database for querying only. Two independent
// guarantees, because this is the surface an agent drives: the file is opened
// read-only, and query_only is set so even a connection that somehow got write
// access refuses to mutate. An agent exploring the data can't damage it.
func OpenReadOnly(path string) (*DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("no database at %s — run `quinto sync` first, or `quinto demo` for sample data", path)
	}

	dsn := "file:" + path + "?mode=ro&_pragma=query_only(1)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	return &DB{DB: sqlDB, Path: path}, nil
}

// Query runs arbitrary SQL and returns its rows. Errors are returned as-is
// from SQLite; the caller is responsible for presenting them readably.
func (db *DB) Query(ctx context.Context, query string) (*Result, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	out := &Result{Columns: cols}
	for rows.Next() {
		scan := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range scan {
			ptrs[i] = &scan[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		// SQLite hands text back as []byte; strings are far more useful to
		// both a terminal and a JSON encoder.
		for i, v := range scan {
			if b, ok := v.([]byte); ok {
				scan[i] = string(b)
			}
		}
		out.Rows = append(out.Rows, scan)
	}
	return out, rows.Err()
}

// Schema returns the database's own DDL, so an agent can discover the shape of
// the data from the CLI instead of reading source or guessing column names.
func (db *DB) Schema(ctx context.Context) (string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT sql FROM sqlite_master
		WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%'
		ORDER BY CASE type WHEN 'table' THEN 0 WHEN 'view' THEN 1 ELSE 2 END, name`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var ddl string
		if err := rows.Scan(&ddl); err != nil {
			return "", err
		}
		parts = append(parts, strings.TrimSpace(ddl)+";")
	}
	return strings.Join(parts, "\n\n"), rows.Err()
}
