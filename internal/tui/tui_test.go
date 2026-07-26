package tui

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Tvk-sd/quinto/internal/demo"
	"github.com/Tvk-sd/quinto/internal/store"
)

func demoModel(t *testing.T) *Model {
	t.Helper()

	db, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	opt := demo.Defaults()
	opt.EndsAt = time.Now()
	if _, _, err := demo.Generate(context.Background(), db, opt); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := db.RecordSync(context.Background(), 0, time.Now(), 0); err != nil {
		t.Fatalf("RecordSync: %v", err)
	}

	m, err := New(db, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m.width, m.height = 96, 24
	return m
}

// streamModel navigates to the stream. The app opens on the overview, so a
// test about the visit list has to say so rather than assume it.
func streamModel(t *testing.T) *Model {
	t.Helper()
	updated, _ := demoModel(t).Update(tea.KeyPressMsg{Code: tea.KeyTab})
	return updated.(*Model)
}

// The first screen is a decision, not an accident: a raw visit list gives a
// first-time reader no context.
func TestOpensOnTheOverview(t *testing.T) {
	if s := demoModel(t).screen; s != screenOverview {
		t.Errorf("screen = %v, want the overview", s)
	}
}

func TestRendersHeaderAndVisits(t *testing.T) {
	m := streamModel(t)
	out := m.render()

	for _, want := range []string{"quinto", "DEMO DATA", "visits", "expand", "quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q", want)
		}
	}
	if !strings.Contains(out, "▶") {
		t.Error("expected collapsed visit markers")
	}
}

// Bots are hidden by default but the count is stated, so the exclusion is
// visible rather than silent.
func TestBotsHiddenByDefaultButDisclosed(t *testing.T) {
	m := demoModel(t)

	if !strings.Contains(m.render(), "bot visits hidden") {
		t.Error("header should disclose that bot visits are hidden")
	}
	for _, s := range m.sessions {
		if s.Bot > 0 {
			t.Fatal("a bot session is in the default list")
		}
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	m = updated.(*Model)

	var bots int
	for _, s := range m.sessions {
		if s.Bot > 0 {
			bots++
		}
	}
	if bots == 0 {
		t.Error("pressing b should bring bot visits into the list")
	}
	if !strings.Contains(m.render(), "bot visits shown") {
		t.Error("header should say bots are shown")
	}
}

func TestExpandingShowsTheJourney(t *testing.T) {
	m := streamModel(t)

	// Find a visit with several pages — that's the case the view exists for.
	for i, s := range m.sessions {
		if s.PageCount >= 3 {
			m.cursor = i
			break
		}
	}
	linesBefore, _, _ := m.buildLines()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(*Model)

	if !strings.Contains(m.render(), "▼") {
		t.Error("expanded visit should show the open marker")
	}

	hits := m.hits[m.sessions[m.cursor].ID]
	if len(hits) == 0 {
		t.Fatal("expanding should load the visit's hits")
	}

	linesAfter, _, _ := m.buildLines()
	if len(linesAfter) <= len(linesBefore) {
		t.Error("expanding should add rows for the journey")
	}

	// The journey's paths must actually be on screen.
	view := m.render()
	if !strings.Contains(view, hits[0].Path) {
		t.Errorf("expanded view should show the first step %q", hits[0].Path)
	}

	// And collapsing puts it back.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(*Model)
	if collapsed, _, _ := m.buildLines(); len(collapsed) != len(linesBefore) {
		t.Error("collapsing should restore the previous row count")
	}
}

// Space is an alias for enter, and it is the one binding the Bubble Tea v2
// upgrade had to rewrite: v1 reported the key as " ", v2 reports it as "space".
// The help text promises it, so it gets a test.
func TestSpaceExpandsLikeEnter(t *testing.T) {
	m := streamModel(t)

	before, _, _ := m.buildLines()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	m = updated.(*Model)

	if after, _, _ := m.buildLines(); len(after) <= len(before) {
		t.Error("space should expand the selected visit, same as enter")
	}
	if !strings.Contains(m.render(), "▼") {
		t.Error("space-expanded visit should show the open marker")
	}
}

// Expanding a visit far down the list must scroll its journey into view.
// Following only the cursor line leaves the steps below the fold — which is
// precisely what the reader just asked to see.
func TestExpandingScrollsTheJourneyIntoView(t *testing.T) {
	m := streamModel(t)
	m.height = 20

	// Pick a deep visit well past the first screenful.
	target := -1
	for i := 25; i < len(m.sessions); i++ {
		if m.sessions[i].PageCount >= 4 {
			target = i
			break
		}
	}
	if target < 0 {
		t.Skip("no deep visit far enough down the demo data")
	}
	m.cursor = target

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(*Model)

	hits := m.hits[m.sessions[target].ID]
	view := m.render()

	var shown int
	for _, h := range hits {
		if strings.Contains(view, h.CreatedAt.Local().Format("15:04:05")) {
			shown++
		}
	}
	if shown == 0 {
		t.Fatalf("expanded journey is entirely off screen:\n%s", view)
	}
	if shown < len(hits) && shown < 5 {
		t.Errorf("only %d of %d journey steps visible", shown, len(hits))
	}
}

func TestNavigationStaysInBounds(t *testing.T) {
	m := streamModel(t)

	for i := 0; i < len(m.sessions)+20; i++ {
		updated, _ := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
		m = updated.(*Model)
	}
	if m.cursor != len(m.sessions)-1 {
		t.Errorf("cursor = %d, want %d", m.cursor, len(m.sessions)-1)
	}

	for i := 0; i < len(m.sessions)+20; i++ {
		updated, _ := m.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
		m = updated.(*Model)
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

// The view must fit its terminal. A line that overflows wraps and destroys
// the layout, and narrow terminals are exactly where a TUI gets judged.
func TestLinesFitNarrowTerminals(t *testing.T) {
	for _, width := range []int{40, 60, 80, 120} {
		m := streamModel(t)
		m.width = width
		m.cursor = 0
		updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = updated.(*Model)

		for i, line := range strings.Split(m.render(), "\n") {
			if got := len([]rune(stripANSI(line))); got > width {
				t.Errorf("width %d: line %d is %d runes: %q", width, i, got, line)
			}
		}
	}
}

// An unmeasurable duration must never render as a number.
func TestUnmeasurableDurationRendersAsDash(t *testing.T) {
	if got := formatDuration(sql.NullInt64{}); got != "—" {
		t.Errorf("formatDuration(NULL) = %q, want an em dash", got)
	}
	if got := formatDuration(sql.NullInt64{Int64: 0, Valid: true}); got != "0s" {
		t.Errorf("a measured zero should still print as 0s, got %q", got)
	}
}

// A NULL referrer means "we never synced the entry hit", which is not the
// same as someone typing the URL. The view must not merge the two.
func TestUnknownReferrerIsNotShownAsDirect(t *testing.T) {
	if got := referrerLabel(sql.NullString{}); got == "direct" {
		t.Error("an unknown referrer must not be labelled direct")
	}
	if got := referrerLabel(sql.NullString{String: "", Valid: true}); got != "direct" {
		t.Errorf("an empty referrer is direct, got %q", got)
	}
}

func TestEmptyDatabaseExplainsWhatToDo(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	m, err := New(db, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m.width, m.height = 80, 24
	// The advice to run sync lives on the stream's empty state.
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(*Model)

	out := m.render()
	if !strings.Contains(out, "sync") || !strings.Contains(out, "demo") {
		t.Errorf("an empty view should point at sync and demo:\n%s", out)
	}
}

// stripANSI removes styling so width assertions measure visible characters.
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEscape = true
		case inEscape && (r == 'm' || r == 'K'):
			inEscape = false
		case !inEscape:
			b.WriteRune(r)
		}
	}
	return b.String()
}
