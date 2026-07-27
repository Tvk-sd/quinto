package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSiteConfig sets up a config file with a default site and a [mctimey]
// named section, sandboxed under a fresh XDG_CONFIG_HOME.
func writeSiteConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "quinto"), 0o755); err != nil {
		t.Fatal(err)
	}
	contents := "site  = tillvonkrueger.goatcounter.com\ntoken = abc123\n\n" +
		"[mctimey]\nsite  = mctimey.goatcounter.com\ntoken = def456\n"
	if err := os.WriteFile(filepath.Join(dir, "quinto", "config"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	f()

	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

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

// --site takes its value as a separate argument, the same fix --db needed —
// and it resolves to that site's own database file, named after it.
func TestSiteFlagAcceptsASeparateValueAndSelectsItsOwnDB(t *testing.T) {
	writeSiteConfig(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	for _, args := range [][]string{
		{"path", "--site", "mctimey"},
		{"--site", "mctimey", "path"},
		{"path", "--site=mctimey"},
	} {
		var out string
		var runErr error
		out = captureStdout(t, func() { runErr = run(args) })
		if runErr != nil {
			t.Errorf("run(%q): %v", args, runErr)
			continue
		}
		if !strings.HasSuffix(strings.TrimSpace(out), filepath.Join("quinto", "mctimey.db")) {
			t.Errorf("run(%q) printed path %q, want it to end in quinto/mctimey.db", args, out)
		}
	}
}

// The default site is addressable by its derived name too — bare `quinto`
// and `--site tillvonkrueger` resolve to the same database.
func TestDefaultSiteIsSelectableByItsDerivedName(t *testing.T) {
	writeSiteConfig(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	bare := captureStdout(t, func() {
		if err := run([]string{"path"}); err != nil {
			t.Fatal(err)
		}
	})
	named := captureStdout(t, func() {
		if err := run([]string{"path", "--site", "tillvonkrueger"}); err != nil {
			t.Fatal(err)
		}
	})
	if bare != named {
		t.Errorf("path = %q, path --site tillvonkrueger = %q — should match", bare, named)
	}
}

// An unrecognised --site fails before any command runs — not just before a
// sync, but before a read-only command like `path` would otherwise silently
// resolve to an empty database nobody configured.
func TestUnknownSiteErrorsBeforeTouchingAnyFile(t *testing.T) {
	writeSiteConfig(t)
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	err := run([]string{"path", "--site", "nope"})
	if err == nil {
		t.Fatal("expected an error for an unconfigured site")
	}
	if !strings.Contains(err.Error(), "tillvonkrueger") || !strings.Contains(err.Error(), "mctimey") {
		t.Errorf("error %q does not name the configured sites", err.Error())
	}

	entries, rerr := os.ReadDir(filepath.Join(dataDir, "quinto"))
	if rerr == nil && len(entries) != 0 {
		t.Errorf("expected no database files to be created, found %v", entries)
	}
}
