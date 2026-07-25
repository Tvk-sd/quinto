package demo

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Tvk-sd/quinto/internal/store"
)

func generate(t *testing.T, opt Options) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "demo.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, _, err := Generate(context.Background(), db, opt); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return db
}

func testOptions() Options {
	o := Defaults()
	// Fixed end date so tests don't drift with the calendar.
	o.EndsAt = time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	return o
}

// Same seed, same data — otherwise the demo GIF and the screenshots in the
// README would drift apart from what a reader gets when they run it.
func TestGenerationIsReproducible(t *testing.T) {
	keysFor := func() []string {
		db := generate(t, testOptions())
		res, err := db.Query(context.Background(), `SELECT hit_key FROM hits ORDER BY hit_key`)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		out := make([]string, len(res.Rows))
		for i, r := range res.Rows {
			out[i], _ = r[0].(string)
		}
		return out
	}

	a, b := keysFor(), keysFor()
	if len(a) == 0 {
		t.Fatal("generated nothing")
	}
	if len(a) != len(b) {
		t.Fatalf("run lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("run differs at %d: %q vs %q", i, a[i], b[i])
		}
	}
}

// Regenerating must not double the dataset — keys are derived from the seed,
// so a second run inserts nothing.
func TestRegeneratingIsANoOp(t *testing.T) {
	db := generate(t, testOptions())
	ctx := context.Background()

	before := count(t, db, `SELECT count(*) FROM hits`)
	if _, inserted, err := Generate(ctx, db, testOptions()); err != nil {
		t.Fatalf("second Generate: %v", err)
	} else if inserted != 0 {
		t.Errorf("second run inserted %d hits, want 0", inserted)
	}
	if after := count(t, db, `SELECT count(*) FROM hits`); after != before {
		t.Errorf("hits went from %d to %d", before, after)
	}
}

// The demo must not flatter the tool. Real small-site traffic is mostly
// bounces and heavily botted; a demo that hides that is a sales pitch.
func TestDataIsRealisticallyUnflattering(t *testing.T) {
	db := generate(t, testOptions())

	total := count(t, db, `SELECT count(*) FROM sessions`)
	bots := count(t, db, `SELECT count(*) FROM sessions WHERE bot > 0`)
	if share := float64(bots) / float64(total); share < 0.15 || share > 0.45 {
		t.Errorf("bot share = %.2f, want a realistic 0.15–0.45", share)
	}

	singles := count(t, db, `SELECT count(*) FROM sessions WHERE bot = 0 AND page_count = 1`)
	humans := count(t, db, `SELECT count(*) FROM sessions WHERE bot = 0`)
	if share := float64(singles) / float64(humans); share < 0.4 || share > 0.8 {
		t.Errorf("single-page share = %.2f, want most visits to bounce (0.4–0.8)", share)
	}

	// Deep sessions must exist, or the expandable stream view has nothing to
	// demonstrate — which is half the reason this dataset exists.
	if deep := count(t, db, `SELECT count(*) FROM sessions WHERE page_count >= 4`); deep < 5 {
		t.Errorf("only %d sessions with 4+ pages — too few to show a journey", deep)
	}

	if events := count(t, db, `SELECT count(*) FROM hits WHERE is_event = 1`); events < 20 {
		t.Errorf("only %d events — real sites fire them, so the demo should too", events)
	}
}

// Flat traffic is the signature of bots. Human traffic must have a day/night
// curve, or anyone who reads analytics will spot the fake immediately.
func TestHumanTrafficHasADiurnalCurve(t *testing.T) {
	db := generate(t, testOptions())

	res, err := db.Query(context.Background(), `
		SELECT CAST(substr(created_at, 12, 2) AS INTEGER) AS hour, count(*)
		FROM hits WHERE bot = 0 GROUP BY 1`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	var night, day int64
	for _, r := range res.Rows {
		hour, _ := r[0].(int64)
		n, _ := r[1].(int64)
		if hour >= 1 && hour <= 5 {
			night += n
		}
		if hour >= 10 && hour <= 16 {
			day += n
		}
	}
	if night == 0 {
		t.Error("no overnight traffic at all — too clean to be believable")
	}
	if day < night*4 {
		t.Errorf("daytime %d vs overnight %d — curve is too flat to look human", day, night)
	}
}

// Journeys must look like journeys: a visit's pages should be connected, not
// independent random draws.
func TestMultiPageVisitsFollowPlausiblePaths(t *testing.T) {
	db := generate(t, testOptions())

	res, err := db.Query(context.Background(), `
		SELECT entry_path, count(*) FROM sessions
		WHERE bot = 0 AND page_count >= 3 GROUP BY 1 ORDER BY 2 DESC`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res.Rows) == 0 {
		t.Fatal("no multi-page sessions")
	}
	// The homepage should dominate entries, as on any real site.
	if top, _ := res.Rows[0][0].(string); top != "/" {
		t.Errorf("most common entry is %q, expected the homepage", top)
	}
}

func count(t *testing.T, db *store.DB, query string) int64 {
	t.Helper()
	res, err := db.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	n, _ := res.Rows[0][0].(int64)
	return n
}
