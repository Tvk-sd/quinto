package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tvk-sd/quinto/internal/store"
)

func TestShouldShowPicker(t *testing.T) {
	cases := []struct {
		name            string
		isDemo          bool
		dbPath, site    string
		configuredSites int
		want            bool
	}{
		{"several sites, bare invocation", false, "", "", 3, true},
		{"exactly one site configured", false, "", "", 1, false},
		{"no sites configured (env-only)", false, "", "", 0, false},
		{"--db overrides even with several sites", false, "/tmp/x.db", "", 3, false},
		{"--site overrides even with several sites", false, "", "mctimey", 3, false},
		{"--demo overrides even with several sites", true, "", "", 3, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldShowPicker(c.isDemo, c.dbPath, c.site, c.configuredSites); got != c.want {
				t.Errorf("shouldShowPicker(%v, %q, %q, %d) = %v, want %v",
					c.isDemo, c.dbPath, c.site, c.configuredSites, got, c.want)
			}
		})
	}
}

func TestSiteRowNeverSyncedWhenFileMissing(t *testing.T) {
	row := siteRow("ghost", filepath.Join(t.TempDir(), "does-not-exist.db"))
	if row.LastSynced != "never synced" {
		t.Errorf("LastSynced = %q, want %q", row.LastSynced, "never synced")
	}
	if row.Visits != 0 || row.Pageviews != 0 {
		t.Errorf("expected zero totals for a missing file, got %+v", row)
	}
}

func TestSiteRowReadsTotalsAndSyncState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	syncedAt := time.Now().Add(-5 * time.Minute)
	if err := db.RecordSync(context.Background(), 42, syncedAt, 0); err != nil {
		t.Fatal(err)
	}
	db.Close()

	row := siteRow("mysite", path)
	if row.Name != "mysite" {
		t.Errorf("Name = %q, want mysite", row.Name)
	}
	if row.LastSynced == "never synced" {
		t.Error("a synced db should not report never synced")
	}
}

func TestSiteRowsIncludesDemoOnlyWhenItExists(t *testing.T) {
	writeSiteConfig(t)
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	rows, err := siteRows()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.IsDemo {
			t.Fatalf("demo row appeared without quinto-demo.db existing: %+v", rows)
		}
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 configured sites (tillvonkrueger, mctimey), got %+v", rows)
	}

	demoPath, err := store.DefaultPath("quinto-demo.db")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Open(demoPath); err != nil {
		t.Fatal(err)
	}

	rows, err = siteRows()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		if r.IsDemo {
			found = true
			if r.Name != "quinto-demo" || r.LastSynced != "demo data" {
				t.Errorf("demo row = %+v, want Name=quinto-demo LastSynced=\"demo data\"", r)
			}
		}
	}
	if !found {
		t.Error("demo row did not appear once quinto-demo.db existed")
	}
}

func TestSiteDBPathResolvesDefaultAndNamed(t *testing.T) {
	writeSiteConfig(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	defaultPath, err := siteDBPath("tillvonkrueger")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(defaultPath) != "quinto.db" {
		t.Errorf("default site path = %q, want quinto.db", defaultPath)
	}

	namedPath, err := siteDBPath("mctimey")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(namedPath) != "mctimey.db" {
		t.Errorf("named site path = %q, want mctimey.db", namedPath)
	}

	if _, err := siteDBPath("nope"); err == nil {
		t.Error("expected an error for an unconfigured site")
	}
}
