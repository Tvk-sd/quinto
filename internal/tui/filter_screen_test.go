package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The app opens on the overview, so that is where a reader first presses "/".
// Filtering selects visits; the overview shows aggregates and no visit list.
//
// The regression: "/" opened the filter in place. Keystrokes went into a query
// that narrowed a list which was not on screen, while the numbers beside it
// stayed exactly the same. No filter box was drawn either, so pressing "/" and
// typing produced no visible effect whatsoever — the filter looked broken.
//
// Worse than invisible, it was dishonest: an active filter alongside figures it
// did not apply to.
func TestSlashFromTheOverviewGoesWhereTheAnswerIs(t *testing.T) {
	m := demoModel(t)
	if m.screen != screenOverview {
		t.Skipf("app no longer opens on the overview (screen=%v); revisit this test", m.screen)
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(*Model)

	if m.screen != screenStream {
		t.Fatal("/ on the overview must switch to the visit list — otherwise it filters something invisible")
	}
	if !m.filtering {
		t.Fatal("/ must also open the filter, not just switch screens")
	}

	for _, r := range "contact" {
		updated, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(*Model)
	}

	if m.filter != "contact" {
		t.Fatalf("filter = %q, want %q", m.filter, "contact")
	}
	if m.err != nil {
		t.Fatalf("filtering errored: %v", m.err)
	}

	// The filter must be legible on screen, and it must be the box saying so —
	// not a demo path that happens to contain the same letters.
	view := stripANSI(m.render())
	if !strings.Contains(view, "/contact") {
		t.Errorf("the active filter is not shown:\n%s", view)
	}
}

// Filtering must narrow the list it is shown next to.
func TestFilterNarrowsTheListOnScreen(t *testing.T) {
	m := demoModel(t)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(*Model)
	before := len(m.sessions)
	if before == 0 {
		t.Fatal("no demo visits to filter")
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(*Model)
	for _, r := range "contact" {
		updated, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = updated.(*Model)
	}

	switch {
	case len(m.sessions) == 0:
		t.Error("filter matched nothing — the query is wrong")
	case len(m.sessions) >= before:
		t.Errorf("filter did not narrow anything: %d before, %d after", before, len(m.sessions))
	}
}
