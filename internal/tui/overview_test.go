package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Tvk-sd/quinto/internal/store"
)

func overviewModel(t *testing.T) *Model {
	t.Helper()
	// The app already opens here; no navigation needed.
	return demoModel(t)
}

func TestOverviewShowsHeadlineNumbers(t *testing.T) {
	m := overviewModel(t)
	out := m.render()

	for _, want := range []string{"visitors", "pageviews", "events", "single-page",
		"PAGE", "REFERRER", "COUNTRY", "pageviews per day"} {
		if !strings.Contains(out, want) {
			t.Errorf("overview is missing %q", want)
		}
	}
}

// The overview's own bot label must agree with the header's — otherwise
// pressing b makes the screen contradict itself: "bot visits shown" up top,
// "bots hidden" in the stat row, at the same time.
func TestOverviewBotsLabelTracksShowBots(t *testing.T) {
	m := overviewModel(t)
	if strings.Contains(m.render(), "bots shown") {
		t.Error("bots are hidden by default; should not say \"bots shown\" yet")
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	m = updated.(*Model)
	out := m.render()
	if !strings.Contains(out, "bots shown") {
		t.Error("after pressing b, the stat row should say \"bots shown\"")
	}
	if strings.Contains(out, "bots hidden") {
		t.Error("after pressing b, the stat row should not still say \"bots hidden\"")
	}
}

// Bounce is reported as a count over its denominator, not a percentage. At
// low traffic a rate implies precision the sample cannot support.
func TestBounceIsShownWithItsDenominator(t *testing.T) {
	m := overviewModel(t)
	out := m.render()

	if strings.Contains(out, "bounce rate") || strings.Contains(out, "%") {
		t.Error("overview should not present a bounce percentage")
	}
	if !strings.Contains(out, "/") {
		t.Error("single-page count should be shown over its denominator")
	}
}

// The range is shared, so switching screens never silently changes what
// period the numbers describe.
func TestRangeIsSharedBetweenScreens(t *testing.T) {
	m := overviewModel(t)
	if !strings.Contains(m.render(), "7 days") {
		t.Fatalf("expected the default range in the overview:\n%s", m.render())
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m = updated.(*Model)
	if m.rng != Range28 {
		t.Errorf("range = %v, want 28 days", m.rng)
	}

	// Switch to the stream and back; the range must survive.
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(*Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = updated.(*Model)

	if !strings.Contains(m.render(), "28 days") {
		t.Error("the range should survive a screen switch")
	}
}

func TestRangeCyclesBackToStart(t *testing.T) {
	r := Range7
	for i := 0; i < 3; i++ {
		r = r.next()
	}
	if r != Range7 {
		t.Errorf("cycling three times gave %v, want the original", r)
	}
}

// A day with no traffic must not draw a bar. Rendering a baseline block for
// an empty day implies activity that did not happen.
func TestEmptyDaysDrawNothing(t *testing.T) {
	m := overviewModel(t)

	day := func(n int, offset int) store.DayCount {
		return store.DayCount{Day: time.Now().AddDate(0, 0, -offset), N: n}
	}
	out := m.sparkline([]store.DayCount{day(10, 4), day(0, 3), day(5, 2), day(0, 1), day(8, 0)})

	bar := strings.Split(stripANSI(out), "\n")[0]
	if !strings.Contains(bar, " ") {
		t.Errorf("empty days should render as blanks, got %q", bar)
	}
	for _, block := range []rune("▁▂▃▄▅▆▇█") {
		if strings.Count(bar, string(block)) > 3 {
			t.Errorf("too many bars for 3 days with data: %q", bar)
		}
	}
}

// With almost no data the chart must say so rather than presenting noise as
// a trend line.
func TestSparseDataIsLabelledAsSparse(t *testing.T) {
	m := overviewModel(t)

	out := m.sparkline([]store.DayCount{
		{Day: time.Now(), N: 3},
		{Day: time.Now().AddDate(0, 0, -1), N: 1},
	})
	if !strings.Contains(out, "too few days") {
		t.Errorf("sparse data should be labelled, got:\n%s", out)
	}

	// And a healthy range should not carry the warning.
	var many []store.DayCount
	for i := 0; i < 14; i++ {
		many = append(many, store.DayCount{Day: time.Now().AddDate(0, 0, -i), N: 5 + i})
	}
	if strings.Contains(m.sparkline(many), "too few days") {
		t.Error("a full range should not be labelled sparse")
	}
}

func TestOverviewFitsNarrowTerminals(t *testing.T) {
	for _, width := range []int{40, 60, 80, 120} {
		m := overviewModel(t)
		m.width = width

		for i, line := range strings.Split(m.render(), "\n") {
			if got := len([]rune(strings.TrimRight(stripANSI(line), " "))); got > width {
				t.Errorf("width %d: line %d is %d runes: %q", width, i, got, line)
			}
		}
	}
}

func TestOverviewOnEmptyRangeSaysSo(t *testing.T) {
	m := overviewModel(t)
	if got := m.sparkline(nil); !strings.Contains(got, "no data") {
		t.Errorf("an empty series should say so, got %q", got)
	}
	if got := m.sparkline([]store.DayCount{{Day: time.Now(), N: 0}}); !strings.Contains(got, "no pageviews") {
		t.Errorf("a range with zero traffic should say so, got %q", got)
	}
}
