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
	//
	// The bots label tracks m.showBots rather than always saying "hidden" —
	// otherwise pressing b contradicts itself: the header already says
	// "bot visits shown" while this row still claimed they were hidden.
	botsLabel := "bots hidden"
	if m.showBots {
		botsLabel = "bots shown"
	}
	b.WriteString(m.statRow([][2]string{
		{"visitors", fmt.Sprint(o.Visitors)},
		{"pageviews", fmt.Sprint(o.Pageviews)},
		{"events", fmt.Sprint(o.Events)},
		{"single-page", fmt.Sprintf("%d/%d", o.SinglePageVisits, o.Visitors)},
		{botsLabel, fmt.Sprint(o.Bots)},
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
		"  tab stream · / filter · r range · b bots · ? help · q quit",
		"  tab · / filter · r range · b · ? · q",
		"  tab · / · r · b · ? · q",
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

// columns renders top pages, referrers and countries as three small tables —
// header row, rule, aligned rows — side by side, falling back to one column
// per row when the terminal is narrow.
func (m *Model) columns(o store.Overview) string {
	type panel struct {
		title string
		items []store.Count
	}
	panels := []panel{
		{"page", o.TopPages},
		{"referrer", o.TopReferrers},
		{"country", o.TopCountries},
	}

	const colWidth = 30
	stacked := m.width < colWidth*2+4

	var b strings.Builder
	if stacked {
		for _, p := range panels {
			for _, line := range panelLines(p.title, p.items, m.width-4, len(p.items)) {
				b.WriteString("  " + line + "\n")
			}
			b.WriteString("\n")
		}
		return b.String()
	}

	perRow := min(len(panels), max(1, (m.width-2)/colWidth))
	for start := 0; start < len(panels); start += perRow {
		end := min(len(panels), start+perRow)
		group := panels[start:end]

		maxRows := 0
		for _, p := range group {
			maxRows = max(maxRows, len(p.items))
		}

		blocks := make([][]string, len(group))
		for i, p := range group {
			blocks[i] = panelLines(p.title, p.items, colWidth, maxRows)
		}
		b.WriteString(joinBlocks(blocks))
	}
	return b.String()
}

// panelLines renders one "label / N" table: a header row, a rule, then
// exactly rows lines — blank-padded past the panel's own item count so it
// lines up with taller neighbours when joined side by side. width is the
// panel's whole slot, margin included, so joinBlocks can concatenate blocks
// with no separator and still leave a gap between panels.
//
// Every line is padded to width as plain text before a style ever touches
// it. Style codes are runes too, so padding a styled string would count
// escape bytes as visible characters and misalign the column next to it —
// the same trap dataLine's own comment warns about, one level up.
func panelLines(title string, items []store.Count, width, rows int) []string {
	const nWidth = 5
	cols := []column{{title, max(4, width-nWidth-2), false}, {"n", nWidth, true}}

	lines := []string{
		header.Render(padTo(headerLine(cols), width)),
		dim.Render(padTo(ruleLine(cols), width)),
	}
	for i := range rows {
		if i < len(items) {
			lines = append(lines, padTo(dataLine(cols, []string{items[i].Label, fmt.Sprint(items[i].N)}), width))
		} else {
			lines = append(lines, strings.Repeat(" ", width))
		}
	}
	if len(items) == 0 {
		lines = append(lines, dim.Render(padTo("—", width)))
	}
	return lines
}

// joinBlocks concatenates same-height line blocks side by side. Each block
// is already padded to its own full slot width (margin included), so this
// is plain string concatenation — no width recomputation, nothing to get
// wrong by counting a style code as a column.
func joinBlocks(blocks [][]string) string {
	rows := 0
	for _, blk := range blocks {
		rows = max(rows, len(blk))
	}
	var b strings.Builder
	for i := range rows {
		var cells []string
		for _, blk := range blocks {
			if i < len(blk) {
				cells = append(cells, blk[i])
			}
		}
		b.WriteString("  " + strings.Join(cells, "") + "\n")
	}
	return b.String()
}

// column, headerLine, ruleLine and dataLine build a small aligned table —
// header row, rule, plain data rows — shared by the overview's three panels
// and the stream view's visit list, so both screens read as one system
// rather than two different layouts that happen to sit in the same app.
type column struct {
	title string
	width int
	right bool // right-align values; used for numeric columns
}

func headerLine(cols []column) string {
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = alignCell(c, strings.ToUpper(c.title))
	}
	return strings.Join(cells, " ")
}

func ruleLine(cols []column) string {
	w := 0
	for i, c := range cols {
		w += c.width
		if i > 0 {
			w++ // the space between columns
		}
	}
	return strings.Repeat("─", w)
}

// dataLine returns plain text on purpose, same reason countLine used to:
// styling per cell here would embed escape codes that alignCell then counts
// as visible characters, quietly misaligning every column.
func dataLine(cols []column, cells []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = alignCell(c, cells[i])
	}
	return strings.Join(out, " ")
}

func alignCell(c column, s string) string {
	if c.right {
		return padLeft(s, c.width)
	}
	return padTo(s, c.width)
}

// padTo truncates before padding — a cell longer than its column would
// otherwise overflow the fixed width every row above and below it commits to.
func padTo(s string, width int) string {
	s = truncate(s, width)
	if n := len([]rune(s)); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}

func padLeft(s string, width int) string {
	s = truncate(s, width)
	if n := len([]rune(s)); n < width {
		return strings.Repeat(" ", width-n) + s
	}
	return s
}

// lipglossBold and lipglossWidth keep the styling imports local to this file.
func lipglossBold() lipgloss.Style { return lipgloss.NewStyle().Bold(true) }
func lipglossWidth(s string) int   { return lipgloss.Width(s) }
