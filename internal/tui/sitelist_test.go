package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func testRows() []SiteRow {
	return []SiteRow{
		{Name: "tillvonkrueger", Visits: 286, Pageviews: 647, LastSynced: "4m ago"},
		{Name: "mctimey", Visits: 12, Pageviews: 31, LastSynced: "2d ago"},
		{Name: "sidegig", Visits: 3, Pageviews: 5, LastSynced: "never synced"},
		{Name: "quinto-demo", Visits: 420, Pageviews: 781, LastSynced: "demo data", IsDemo: true},
	}
}

func pressKey(m *Picker, key string) *Picker {
	updated, _ := m.Update(tea.KeyPressMsg{Text: key, Code: rune(key[0])})
	return updated.(*Picker)
}

func TestPickerRendersEverySiteAndTheBanner(t *testing.T) {
	m := NewPicker(testRows())
	out := m.render()
	for _, want := range []string{"Q U I N T O", "tillvonkrueger", "mctimey", "sidegig", "quinto-demo", "never synced"} {
		if !strings.Contains(out, want) {
			t.Errorf("view is missing %q", want)
		}
	}
}

func TestPickerCursorMovesWithinBounds(t *testing.T) {
	m := NewPicker(testRows())
	if m.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", m.cursor)
	}

	m = pressKey(m, "k") // up at the top is a no-op
	if m.cursor != 0 {
		t.Errorf("cursor moved above the top: %d", m.cursor)
	}

	for range len(testRows()) + 2 { // deliberately overshoot the bottom
		m = pressKey(m, "j")
	}
	if want := len(testRows()) - 1; m.cursor != want {
		t.Errorf("cursor = %d, want clamped to %d", m.cursor, want)
	}
}

func TestEnterOpensTheSelectedSite(t *testing.T) {
	m := NewPicker(testRows())
	m = pressKey(m, "j") // -> mctimey
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "enter", Code: tea.KeyEnter})
	m = updated.(*Picker)

	if cmd == nil {
		t.Fatal("enter should quit the program")
	}
	if got := m.Result(); got.Action != ActionOpen || got.Site != "mctimey" {
		t.Errorf("Result() = %+v, want ActionOpen on mctimey", got)
	}
}

func TestLowercaseSSyncsTheSelectedSite(t *testing.T) {
	m := NewPicker(testRows())
	m = pressKey(m, "s")
	if got := m.Result(); got.Action != ActionSync || got.Site != "tillvonkrueger" {
		t.Errorf("Result() = %+v, want ActionSync on tillvonkrueger", got)
	}
}

// The demo row isn't a configured site — `quinto sync` has nothing to fetch
// for it, so pressing s there must not produce a sync attempt at all.
func TestLowercaseSOnTheDemoRowDoesNothing(t *testing.T) {
	rows := testRows()
	m := NewPicker(rows)
	for range len(rows) - 1 {
		m = pressKey(m, "j")
	}
	if !rows[m.cursor].IsDemo {
		t.Fatalf("test setup: cursor should be on the demo row, got %q", rows[m.cursor].Name)
	}

	updated, cmd := m.Update(tea.KeyPressMsg{Text: "s", Code: 's'})
	m = updated.(*Picker)
	if cmd != nil {
		t.Error("s on the demo row should not quit the program")
	}
	if got := m.Result(); got != (PickerResult{}) {
		t.Errorf("Result() should stay empty, got %+v", got)
	}
}

// S picks the stalest real site regardless of the cursor's position — never
// synced beats any real age, and demo is never a candidate.
func TestCapitalSPicksTheStalestRealSite(t *testing.T) {
	m := NewPicker(testRows())
	updated, _ := m.Update(tea.KeyPressMsg{Text: "S", Code: 'S'})
	m = updated.(*Picker)

	if got := m.Result(); got.Action != ActionSync || got.Site != "sidegig" {
		t.Errorf("Result() = %+v, want ActionSync on sidegig (never synced)", got)
	}
}

func TestCapitalSWithNoRealSitesDoesNothing(t *testing.T) {
	m := NewPicker([]SiteRow{{Name: "quinto-demo", IsDemo: true, LastSynced: "demo data"}})
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "S", Code: 'S'})
	m = updated.(*Picker)

	if cmd != nil {
		t.Error("S with only a demo row should not quit the program")
	}
	if got := m.Result(); got != (PickerResult{}) {
		t.Errorf("Result() should stay empty, got %+v", got)
	}
}

func TestQuitReturnsActionQuit(t *testing.T) {
	m := NewPicker(testRows())
	updated, cmd := m.Update(tea.KeyPressMsg{Text: "q", Code: 'q'})
	m = updated.(*Picker)

	if cmd == nil {
		t.Fatal("q should quit the program")
	}
	if got := m.Result(); got.Action != ActionQuit {
		t.Errorf("Result() = %+v, want ActionQuit", got)
	}
}
