package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Tvk-sd/quinto/internal/store"
)

// filterModel builds a deliberately small, ordered set of visits, on the stream
// screen. Demo data is the wrong fixture here: these tests are about which row
// the cursor lands on, so which rows exist has to be exact.
//
// Newest first, as the stream shows them:
//
//	workA    /work      no bot
//	plain    /          no bot
//	crawler  /robots    BOT — hidden by default
//	keep     /work      no bot, and reached /contact on its second page
func filterModel(t *testing.T) *Model {
	t.Helper()

	db, err := store.Open(filepath.Join(t.TempDir(), "filter.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	h := func(key, session, path, at string, first bool, bot int) store.Hit {
		return store.Hit{
			Key: key, Session: session, Path: path, CreatedAt: at,
			FirstVisit: first, Bot: bot,
			Referrer: "Hacker News", ReferrerScheme: "g",
			Country: "DE", Browser: "Chrome 126", System: "macOS 10.15",
		}
	}
	if _, err := db.InsertHits(context.Background(), []store.Hit{
		h("a1", "workA", "/work", "2026-07-20T14:00:00Z", true, 0),
		h("p1", "plain", "/", "2026-07-20T13:00:00Z", true, 0),
		h("c1", "crawler", "/robots.txt", "2026-07-20T12:00:00Z", true, 1),
		h("k1", "keep", "/work", "2026-07-20T11:00:00Z", true, 0),
		h("k2", "keep", "/contact", "2026-07-20T11:00:30Z", false, 0),
	}); err != nil {
		t.Fatalf("InsertHits: %v", err)
	}

	m, err := New(db, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m.width, m.height = 100, 30
	m.screen = screenStream
	return m
}

func press(t *testing.T, m *Model, keys ...string) (*Model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		default:
			msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
		var updated tea.Model
		updated, cmd = m.Update(msg)
		m = updated.(*Model)
	}
	return m, cmd
}

func visitIDs(m *Model) []string {
	out := make([]string, len(m.sessions))
	for i, s := range m.sessions {
		out[i] = s.ID
	}
	return out
}

// The decision the whole ticket rests on: searching for a page finds the people
// who reached it, not only those who landed on it — and says which page it was,
// because that row shows no sign of it otherwise.
func TestFilterFindsAPageReachedInsideAVisit(t *testing.T) {
	m := filterModel(t)
	m, _ = press(t, m, "/", "c", "o", "n", "t", "a", "c", "t")

	if got := visitIDs(m); len(got) != 1 || got[0] != "keep" {
		t.Fatalf("visits = %v, want just the visit that reached /contact", got)
	}
	if m.sessions[0].Match != "/contact" {
		t.Errorf("Match = %q, want the page that matched", m.sessions[0].Match)
	}
	if out := m.render(); !strings.Contains(out, "/contact") {
		t.Errorf("the row should say which page matched:\n%s", out)
	}
}

// The highlight follows the visit, not the row number. "keep" moves from row 3
// to row 2 when the filter is applied, so a cursor that merely stayed put would
// be visibly wrong.
func TestFilterKeepsTheHighlightOnYourVisit(t *testing.T) {
	m := filterModel(t)

	m, _ = press(t, m, "j", "j") // workA -> plain -> keep
	if m.cursorVisit != "keep" {
		t.Fatalf("setup: cursor is on %q, want keep", m.cursorVisit)
	}

	m, _ = press(t, m, "/", "w", "o", "r", "k")

	if got := visitIDs(m); len(got) != 2 {
		t.Fatalf("visits = %v, want workA and keep", got)
	}
	if m.cursorVisit != "keep" {
		t.Errorf("cursor moved to %q; it should still be on keep", m.cursorVisit)
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1 — keep is the second row now", m.cursor)
	}
}

// When the visit under the cursor stops matching, the highlight goes to the top
// rather than to whatever now occupies the old row number.
//
// The fixture makes those two different rows on purpose: sitting on the bot at
// row 2 and hiding bots, the old row number holds "keep" while the top holds
// "workA". Falling back to the row number would hand the reader an unrelated
// visit they never selected — the behaviour anchoring by visit exists to avoid.
func TestLostAnchorGoesToTheTopNotTheOldRowNumber(t *testing.T) {
	m := filterModel(t)

	m, _ = press(t, m, "b") // show bots: workA, plain, crawler, keep
	m, _ = press(t, m, "j", "j")
	if m.cursorVisit != "crawler" {
		t.Fatalf("setup: cursor is on %q, want the bot visit", m.cursorVisit)
	}
	oldIndex := m.cursor

	m, _ = press(t, m, "b") // hide bots again: the anchored visit is destroyed

	if m.cursor != 0 || m.cursorVisit != "workA" {
		t.Errorf("cursor = %d (%s), want row 0 (workA)", m.cursor, m.cursorVisit)
	}
	if oldIndex < len(m.sessions) && m.cursorVisit == m.sessions[oldIndex].ID {
		t.Errorf("cursor fell back to the old row number (%s) instead of the top",
			m.sessions[oldIndex].ID)
	}
}

// Opening a visit is a request that outlives a filter change.
func TestExpandedVisitsSurviveAFilterChange(t *testing.T) {
	m := filterModel(t)

	m, _ = press(t, m, "j", "j", "enter")
	if !m.expanded["keep"] {
		t.Fatalf("setup: keep should be expanded")
	}

	m, _ = press(t, m, "/", "w", "o", "r", "k")
	if !m.expanded["keep"] {
		t.Error("filtering closed a visit the reader had opened")
	}

	m, _ = press(t, m, "enter", "esc") // commit, then clear
	if !m.expanded["keep"] {
		t.Error("clearing the filter should not close it either")
	}
}

// esc clears before it quits. q and ctrl+c always quit, so nothing is trapped.
func TestEscClearsTheFilterBeforeQuitting(t *testing.T) {
	m := filterModel(t)

	m, _ = press(t, m, "/", "w", "o", "r", "k")
	m, cmd := press(t, m, "enter") // commit, back to browsing
	if cmd != nil {
		t.Fatal("committing a filter should not quit")
	}

	m, cmd = press(t, m, "esc")
	if cmd != nil {
		t.Error("esc with a filter active should clear it, not quit")
	}
	if m.filter != "" {
		t.Errorf("filter = %q, want cleared", m.filter)
	}
	if len(m.sessions) != 3 {
		t.Errorf("clearing should restore the full list, got %v", visitIDs(m))
	}

	if _, cmd = press(t, m, "esc"); cmd == nil {
		t.Error("esc with nothing to clear should quit")
	}
}

// Once the filter box is open every letter is text. "b" is the bots toggle in
// browsing mode and a perfectly ordinary thing to search for.
func TestLettersAreTextWhileTyping(t *testing.T) {
	m := filterModel(t)
	wasShowingBots := m.showBots

	m, _ = press(t, m, "/", "b")

	if m.filter != "b" {
		t.Errorf("filter = %q, want %q — b should have been typed", m.filter, "b")
	}
	if m.showBots != wasShowingBots {
		t.Error("b toggled the bots filter while it was being typed into the box")
	}
}

// A filter emptying the list is a different situation from having no data, and
// the hint for one is useless in the other.
func TestEmptyResultsDoNotOfferAnUnhelpfulBotsHint(t *testing.T) {
	m := filterModel(t)

	// Nothing, bot or otherwise, contains this.
	m, _ = press(t, m, "/", "z", "z", "z", "z")
	out := m.render()

	if !strings.Contains(out, "No visits match") {
		t.Errorf("expected a filter-specific empty state:\n%s", out)
	}
	if strings.Contains(out, "press b") {
		t.Errorf("no hidden bot visit matches, so b would not help:\n%s", out)
	}

	// /robots.txt is only reachable by showing bots, so here the hint earns
	// its place.
	m = filterModel(t)
	m, _ = press(t, m, "/", "r", "o", "b", "o", "t")
	if out := m.render(); !strings.Contains(out, "press b") {
		t.Errorf("a hidden bot visit does match; the hint should say so:\n%s", out)
	}
}
