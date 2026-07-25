package goatcounter

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// The fixture is a real export captured from the live API on 2026-07-25, not
// a hand-authored approximation. Four things in it contradict the published
// docs, and every one would have produced a silently wrong parser.
func TestParseRealExport(t *testing.T) {
	f, err := os.Open("testdata/export-v2.csv")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	hits, err := ParseExport(f)
	if err != nil {
		t.Fatalf("ParseExport: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("parsed %d hits, want 3", len(hits))
	}

	h := hits[0]
	if h.Path != "/quinto-test" {
		t.Errorf("Path = %q", h.Path)
	}
	if h.Title != "quinto diagnostic - entry" {
		t.Errorf("Title = %q", h.Title)
	}
	// GoatCounter normalises referrers: news.ycombinator.com arrives as a
	// display name, with the scheme saying how it was derived.
	if h.Referrer != "Hacker News" || h.ReferrerScheme != "g" {
		t.Errorf("Referrer = %q scheme %q, want \"Hacker News\"/\"g\"", h.Referrer, h.ReferrerScheme)
	}
	// Screen size contains commas and is quoted — a naive split on "," would
	// shift every column after it.
	if !strings.Contains(h.ScreenSize, ",") {
		t.Errorf("ScreenSize = %q, expected a comma-containing value", h.ScreenSize)
	}
	if h.Country != "DE" {
		t.Errorf("Country = %q", h.Country)
	}
	if !h.FirstVisit {
		t.Error("FirstVisit should be true on the first hit")
	}
	if h.Bot != 0 {
		t.Errorf("Bot = %d", h.Bot)
	}
	if h.CreatedAt != "2026-07-25T11:32:39Z" {
		t.Errorf("CreatedAt = %q", h.CreatedAt)
	}
	if h.Session == "" {
		t.Error("Session must be populated — the stream view groups by it")
	}
}

// Every row must get a distinct key, since GoatCounter's export has no id and
// deduplication depends on it.
func TestKeysAreDistinctAndStable(t *testing.T) {
	read := func() []Hit {
		f, err := os.Open("testdata/export-v2.csv")
		if err != nil {
			t.Fatalf("open fixture: %v", err)
		}
		defer f.Close()
		hits, err := ParseExport(f)
		if err != nil {
			t.Fatalf("ParseExport: %v", err)
		}
		return hits
	}

	first, second := read(), read()
	seen := map[string]bool{}
	for i, h := range first {
		if h.Key == "" {
			t.Fatalf("hit %d has no key", i)
		}
		if seen[h.Key] {
			t.Errorf("duplicate key %q", h.Key)
		}
		seen[h.Key] = true
		if second[i].Key != h.Key {
			t.Errorf("key for hit %d is not stable across parses", i)
		}
	}
}

// Their docs are explicit that a consumer should fail loudly on a format
// change rather than mis-parse into a corrupt local database.
func TestUnsupportedVersionIsRefused(t *testing.T) {
	in := strings.NewReader("9Path,Title,Session,Date\n/,Home,abc,2026-07-25T11:32:39Z\n")

	_, err := ParseExport(in)
	var ve *VersionError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want *VersionError", err)
	}
	if ve.Got != 9 || ve.Want != SupportedVersion {
		t.Errorf("got version %d, want %d", ve.Got, ve.Want)
	}
}

func TestHeaderWithoutVersionIsRejected(t *testing.T) {
	in := strings.NewReader("Path,Title,Session,Date\n")
	if _, err := ParseExport(in); err == nil {
		t.Fatal("a header with no version prefix must be rejected")
	}
}

// An export with no new hits is a valid, empty response — "nothing since the
// cursor", not a failure.
func TestEmptyExportIsNotAnError(t *testing.T) {
	hits, err := ParseExport(strings.NewReader(""))
	if err != nil {
		t.Fatalf("empty export: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("parsed %d hits from an empty export", len(hits))
	}
}

// Columns are addressed by name, so a reordered export keeps working rather
// than silently mapping the wrong values.
func TestColumnsAreAddressedByName(t *testing.T) {
	in := strings.NewReader(
		"2Date,Session,Path,Bot,FirstVisit\n" +
			"2026-07-25T11:32:39Z,sess-x,/reordered,1,1\n")

	hits, err := ParseExport(in)
	if err != nil {
		t.Fatalf("ParseExport: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("parsed %d hits", len(hits))
	}
	h := hits[0]
	if h.Path != "/reordered" || h.Session != "sess-x" || h.Bot != 1 || !h.FirstVisit {
		t.Errorf("columns mapped by position, not name: %+v", h)
	}
}

func TestMissingRequiredColumnIsRejected(t *testing.T) {
	in := strings.NewReader("2Title,Browser\nHome,Chrome\n")
	if _, err := ParseExport(in); err == nil {
		t.Fatal("an export without Path/Session/Date must be rejected")
	}
}
