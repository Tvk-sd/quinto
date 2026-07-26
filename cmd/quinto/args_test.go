package main

import (
	"os"
	"path/filepath"
	"testing"
)

// --db takes its value as a separate argument. Sorting arguments by their
// leading dash used to strip the two apart: the flag lost its value and the
// path was treated as a subcommand.
func TestDBFlagAcceptsASeparateValue(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "site2.db")

	for _, args := range [][]string{
		{"demo", "--db", db},
		{"query", "select count(*) from hits", "--db", db},
		{"--db", db, "query", "select count(*) from hits"},
		{"query", "select count(*) from hits", "--db=" + db},
	} {
		if err := run(args); err != nil {
			t.Errorf("run(%q): %v", args, err)
		}
	}

	if _, err := os.Stat(db); err != nil {
		t.Fatalf("the database was not created where --db pointed: %v", err)
	}
}

// Without --db the real database is used, not a stray positional.
func TestDefaultPathIsUsedWithoutTheFlag(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := run([]string{"demo"}); err != nil {
		t.Fatalf("run: %v", err)
	}
}
