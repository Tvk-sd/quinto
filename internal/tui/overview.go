package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Tvk-sd/quinto/internal/store"
)

// Range is the period both screens share, so switching views never silently
// changes what you're looking at.
type Range int

const (
	Range7 Range = iota
	Range28
	RangeAll
)

func (r Range) label() string {
	switch r {
	case Range7:
		return "7 days"
	case Range28:
		return "28 days"
	default:
		return "all time"
	}
}

func (r Range) since() time.Time {
	switch r {
	case Range7:
		return time.Now().AddDate(0, 0, -7)
	case Range28:
		return time.Now().AddDate(0, 0, -28)
	default:
		return time.Time{}
	}
}

func (r Range) next() Range {
	if r == RangeAll {
		return Range7
	}
	return r + 1
}

var (
	statLabel = dim
	statValue = lipglossBold()
)

func (m *Model) overviewView() string {
	o := m.overview

	var b strings.Builder
	b.WriteString(m.headerView())
	b.WriteString("\n\n")

	// Headline numbers. Bounces are shown as a count with its denominator
	// rather than a rate, so a reader can see how thin the base is.
	b.WriteString(m.statRow([][2]string{
		{"visitors", fmt.Sprint(o.Visitors)},
		{"pageviews", fmt.Sprint(o.Pageviews)},
		{"events", fmt.Sprint(o.Events)},
		{"single-page", fmt.Sprintf("%d/%d", o.SinglePageVisits, o.Visitors)},
		{"bots hidden", fmt.Sprint(o.Bots)},
	}))
	b.WriteString("\n\n")

	b.WriteString(dim.Render("  pageviews per day · " + m.rng.label()))
	b.WriteString("\n")
	b.WriteString(m.sparkline(o.Daily))
	b.WriteString("\n\n")

	b.WriteString(m.columns(o))
	b.WriteString("\n")
	b.WriteString(m.overviewFooter())
	return b.String()
}

// overviewFooter shortens rather than wrapping, same as the stream view's.
func (m *Model) overviewFooter() string {
	for _, hints := range []string{
		"  tab stream · r range · b bots · ? help · q quit",
		"  tab · r range · b · ? · q",
		"  ? help · q quit",
		"  ?",
	} {
		if len([]rune(hints)) <= m.width {
			return dim.Render(hints)
		}
	}
	return ""
}

func (m *Model) statRow(stats [][2]string) string {
	var parts []string
	for _, s := range stats {
		parts = append(parts, statValue.Render(s[1])+" "+statLabel.Render(s[0]))
	}

	line := "  " + strings.Join(parts, statLabel.Render("   "))
	if lipglossWidth(line) <= m.width {
		return line
	}
	// Too narrow for one row — stack them instead of wrapping mid-label.
	var b strings.Builder
	for _, s := range stats {
		b.WriteString("  " + statValue.Render(s[1]) + " " + statLabel.Render(s[0]) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// sparkline draws pageviews per day.
//
// Days with no traffic render as a space, never as a baseline block: drawing
// something for a day that had nothing implies activity that did not happen.
// With very few days of data the chart is labelled as such rather than being
// stretched to look like a trend.
func (m *Model) sparkline(days []store.DayCount) string {
	if len(days) == 0 {
		return dim.Render("  no data yet")
	}

	blocks := []rune("▁▂▃▄▅▆▇█")

	maxN := 0
	withData := 0
	for _, d := range days {
		if d.N > maxN {
			maxN = d.N
		}
		if d.N > 0 {
			withData++
		}
	}
	if maxN == 0 {
		return dim.Render("  no pageviews in this range")
	}

	// Keep the most recent days that fit the terminal.
	avail := max(10, m.width-6)
	if len(days) > avail {
		days = days[len(days)-avail:]
	}

	var bar strings.Builder
	for _, d := range days {
		if d.N == 0 {
			bar.WriteRune(' ')
			continue
		}
		idx := (d.N * (len(blocks) - 1)) / maxN
		bar.WriteRune(blocks[idx])
	}

	out := "  " + pathStyle.Render(bar.String())

	scale := fmt.Sprintf("  peak %d/day · %d of %d days had traffic",
		maxN, withData, len(days))
	if withData < 5 {
		scale += " · too few days to call it a trend"
	}
	return out + "\n" + dim.Render(scale)
}

// columns renders top pages, referrers and countries side by side, falling
// back to a single column when the terminal is narrow.
func (m *Model) columns(o store.Overview) string {
	type col struct {
		title string
		items []store.Count
	}
	cols := []col{
		{"top pages", o.TopPages},
		{"referrers", o.TopReferrers},
		{"countries", o.TopCountries},
	}

	const colWidth = 30
	stacked := m.width < colWidth*2+4

	var b strings.Builder
	if stacked {
		for _, c := range cols {
			b.WriteString(dim.Render("  "+c.title) + "\n")
			b.WriteString(m.countList(c.items, m.width-4))
			b.WriteString("\n")
		}
		return b.String()
	}

	perRow := min(len(cols), max(1, (m.width-2)/colWidth))
	for start := 0; start < len(cols); start += perRow {
		end := min(len(cols), start+perRow)
		group := cols[start:end]

		var titles []string
		for _, c := range group {
			titles = append(titles, padTo(c.title, colWidth))
		}
		b.WriteString("  " + dim.Render(strings.Join(titles, "")) + "\n")

		rows := 0
		for _, c := range group {
			rows = max(rows, len(c.items))
		}
		for i := 0; i < rows; i++ {
			var cells []string
			for _, c := range group {
				if i < len(c.items) {
					cells = append(cells, padTo(countLine(c.items[i], colWidth-1), colWidth))
				} else {
					cells = append(cells, strings.Repeat(" ", colWidth))
				}
			}
			b.WriteString("  " + strings.TrimRight(strings.Join(cells, ""), " ") + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m *Model) countList(items []store.Count, width int) string {
	if len(items) == 0 {
		return dim.Render("    —") + "\n"
	}
	var b strings.Builder
	for _, c := range items {
		b.WriteString("    " + countLine(c, width-4) + "\n")
	}
	return b.String()
}

// countLine returns plain text on purpose. Styling here would embed escape
// codes that padTo then counts as visible characters, quietly misaligning
// every column.
func countLine(c store.Count, width int) string {
	n := fmt.Sprint(c.N)
	label := truncate(c.Label, max(3, width-len(n)-2))
	gap := width - len([]rune(label)) - len(n)
	if gap < 1 {
		gap = 1
	}
	return label + strings.Repeat(" ", gap) + n
}

func padTo(s string, width int) string {
	if n := len([]rune(s)); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

// lipglossBold and lipglossWidth keep the styling imports local to this file.
func lipglossBold() lipgloss.Style { return lipgloss.NewStyle().Bold(true) }
func lipglossWidth(s string) int   { return lipgloss.Width(s) }
