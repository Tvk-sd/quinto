package store

import (
	"context"
	"testing"
)

// filterFixture builds two visits whose difference is the point of the whole
// filter design: one *lands* on /contact, the other reaches it on its second
// page and so never shows the word on its row.
func filterFixture(t *testing.T) *DB {
	t.Helper()
	db := openTemp(t)

	insert(t, db,
		hit("h1", "landed", "/contact", "2026-07-20T10:00:00Z", true, 0),

		hit("h2", "reached", "/", "2026-07-20T11:00:00Z", true, 0),
		hit("h3", "reached", "/contact", "2026-07-20T11:00:30Z", false, 0),

		hit("h4", "elsewhere", "/work", "2026-07-20T12:00:00Z", true, 0),

		hit("h5", "crawler", "/contact", "2026-07-20T13:00:00Z", true, 1),
	)
	return db
}

func ids(sessions []Session) []string {
	out := make([]string, len(sessions))
	for i, s := range sessions {
		out[i] = s.ID
	}
	return out
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

// The decision this ticket exists for: a visit matches on a page it reached,
// not only on the page it landed on. Row-only matching finds "landed" and
// misses "reached" — and in the real dataset that was 24 visits missed.
func TestFilterMatchesPagesInsideTheVisit(t *testing.T) {
	db := filterFixture(t)

	got, err := db.RecentSessions(context.Background(), 50, SessionFilter{Query: "contact"})
	if err != nil {
		t.Fatalf("RecentSessions: %v", err)
	}

	found := ids(got)
	for _, want := range []string{"landed", "reached"} {
		if !contains(found, want) {
			t.Errorf("filtering on %q should find the %q visit, got %v", "contact", want, found)
		}
	}
	if contains(found, "elsewhere") {
		t.Errorf("a visit that never saw /contact matched: %v", found)
	}
}

// A visit can match on something its row never displays, so the filter has to
// be able to say why. Rows that match visibly say nothing, because repeating
// what is already on screen is noise.
func TestFilterReportsWhyAVisitMatchedWhenItIsNotVisible(t *testing.T) {
	db := filterFixture(t)

	got, err := db.RecentSessions(context.Background(), 50, SessionFilter{Query: "contact"})
	if err != nil {
		t.Fatalf("RecentSessions: %v", err)
	}

	for _, s := range got {
		switch s.ID {
		case "reached":
			if s.Match != "/contact" {
				t.Errorf("reached: Match = %q, want the step that matched", s.Match)
			}
		case "landed":
			if s.Match != "" {
				t.Errorf("landed: Match = %q, want empty — the row already shows it", s.Match)
			}
		}
	}
}

// Bots stay excluded by default with a filter active, and the toggle still
// lifts the exclusion rather than deleting rows.
func TestFilterComposesWithTheBotsToggle(t *testing.T) {
	db := filterFixture(t)
	ctx := context.Background()

	without, err := db.RecentSessions(ctx, 50, SessionFilter{Query: "contact"})
	if err != nil {
		t.Fatalf("RecentSessions: %v", err)
	}
	if contains(ids(without), "crawler") {
		t.Error("a bot visit matched while bots were hidden")
	}

	with, err := db.RecentSessions(ctx, 50, SessionFilter{Query: "contact", IncludeBots: true})
	if err != nil {
		t.Fatalf("RecentSessions: %v", err)
	}
	if !contains(ids(with), "crawler") {
		t.Error("showing bots should bring the matching bot visit into the list")
	}
}

// Typing a wildcard must filter for that character, not match everything.
// Unescaped, "%" reads as "show me all of it", which looks like the filter is
// broken rather than like it is working.
func TestFilterTreatsWildcardsAsText(t *testing.T) {
	db := filterFixture(t)

	got, err := db.RecentSessions(context.Background(), 50, SessionFilter{Query: "%"})
	if err != nil {
		t.Fatalf("RecentSessions: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("%%  matched %d visits; no path here contains a literal percent", len(got))
	}
}

// An empty filter is not a filter: it must return the same list as before the
// feature existed.
func TestEmptyFilterReturnsEverything(t *testing.T) {
	db := filterFixture(t)

	got, err := db.RecentSessions(context.Background(), 50, SessionFilter{})
	if err != nil {
		t.Fatalf("RecentSessions: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d visits, want the 3 non-bot ones: %v", len(got), ids(got))
	}
	for _, s := range got {
		if s.Match != "" {
			t.Errorf("%s: Match = %q with no filter active", s.ID, s.Match)
		}
	}
}
