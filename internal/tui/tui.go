// Package tui renders the stream view — quinto's main screen.
//
// One row per visit, expandable to the path the visitor took. At the traffic
// levels this tool is built for, the within-session journey is the only story
// the data actually contains: aggregates need volume that most sites don't
// have, but a list of visits is honest at any n.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"

	"github.com/Tvk-sd/quinto/internal/store"
)

const sessionLimit = 500

// adaptive picks a colour by terminal background. Lip Gloss v2 dropped the
// built-in AdaptiveColor; compat keeps the v1 behaviour of detecting the
// background once, globally, which is all this app needs.
func adaptive(light, dark string) compat.AdaptiveColor {
	return compat.AdaptiveColor{Light: lipgloss.Color(light), Dark: lipgloss.Color(dark)}
}

var (
	// dim carries most of the app's actual information (stats, hints,
	// referrer/country detail), so it needs to clear WCAG AA's 4.5:1 body-text
	// contrast against common terminal backgrounds, not just look "subtle."
	// 245/241 measured 2.3-3.5:1; 241/247 clears ~4.5:1+ in both modes.
	dim = lipgloss.NewStyle().Foreground(adaptive("241", "247"))

	header = lipgloss.NewStyle().
		Foreground(adaptive("236", "252")).
		Bold(true)

	demoTag = lipgloss.NewStyle().
		Foreground(adaptive("130", "214")).
		Bold(true)

	// selected marks the current row with a full-width background band
	// rather than coloured text — Till's call, after comparing against
	// claws' table style (2026-07-26). Same accent hue as before (27/117
	// blue), just carried by the background instead of the foreground.
	selected = lipgloss.NewStyle().
			Bold(true).
			Foreground(adaptive("230", "0")).
			Background(adaptive("27", "117"))

	pathStyle = lipgloss.NewStyle().
			Foreground(adaptive("22", "114"))

	eventStyle = lipgloss.NewStyle().
			Foreground(adaptive("97", "141"))

	// botStyle stays a step dimmer than dim (bot rows should recede) but
	// 244/240 measured 2.0-4.0:1 — too low to read at all in several dark
	// themes. 243/245 keeps the recede-relative-to-dim relationship while
	// staying legible (~4.1:1+).
	botStyle = lipgloss.NewStyle().
			Foreground(adaptive("243", "245")).
			Italic(true)

	errStyle = lipgloss.NewStyle().
			Foreground(adaptive("160", "203"))
)

type screen int

const (
	screenStream screen = iota
	screenOverview
)

// Model is the TUI's state, shared by both screens so switching never changes
// what period you are looking at.
type Model struct {
	db     *store.DB
	isDemo bool

	sessions []store.Session
	hits     map[string][]store.SessionHit
	expanded map[string]bool

	cursor   int
	offset   int
	showBots bool
	showHelp bool

	// filter narrows the list; filtering means the reader is still typing it.
	//
	// cursorVisit is what the cursor is *for*: an index into a slice the filter
	// resizes underneath it points at a row number, and a row number is not
	// what anyone is reading. Keeping the id lets the highlight follow the
	// visit instead.
	filter      string
	filtering   bool
	cursorVisit string

	// botsWouldMatch counts hidden bot visits matching the active filter, so
	// the empty state only offers "press b" when pressing b would help.
	botsWouldMatch int

	syncedAt time.Time
	synced   bool
	totals   store.Totals

	screen   screen
	rng      Range
	overview store.Overview

	width, height int
	err           error
}

// New loads the data the view needs. The dataset is small and local, so
// loading synchronously keeps the model simple — there is no spinner to
// justify.
func New(db *store.DB, isDemo bool) (*Model, error) {
	m := &Model{
		db:       db,
		isDemo:   isDemo,
		hits:     map[string][]store.SessionHit{},
		expanded: map[string]bool{},
		width:    80,
		height:   24,
		// Land on the overview first: a raw visit list has no context for a
		// reader who just opened the app. tab still reaches the stream.
		screen: screenOverview,
	}
	if err := m.reload(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Model) reload() error {
	if err := m.reloadSessions(); err != nil {
		return err
	}

	ctx := context.Background()
	var err error
	if m.totals, err = m.db.Totals(ctx); err != nil {
		return err
	}
	if state, err := m.db.SyncState(ctx); err == nil && state.Synced {
		m.syncedAt, m.synced = state.LastSyncedAt, true
	}
	if m.overview, err = m.db.LoadOverview(ctx, m.rng.since()); err != nil {
		return err
	}
	return nil
}

// reloadSessions re-runs just the visit query. Filtering happens in SQL rather
// than over an in-memory slice, so the list on screen is always the result of a
// statement someone could have typed into `quinto query` — the parity this
// project claims is structural rather than maintained by hand.
//
// It is separate from reload because it runs on every keystroke while a filter
// is being typed, and the totals, sync state and overview aggregates do not
// change when the filter does.
func (m *Model) reloadSessions() error {
	ctx := context.Background()
	f := store.SessionFilter{Query: m.filter, IncludeBots: m.showBots}

	sessions, err := m.db.RecentSessions(ctx, sessionLimit, f)
	if err != nil {
		return err
	}
	m.sessions = sessions

	// Only worth asking when the answer could change what the empty state says.
	m.botsWouldMatch = 0
	if len(sessions) == 0 && !m.showBots {
		f.IncludeBots = true
		if withBots, err := m.db.RecentSessions(ctx, sessionLimit, f); err == nil {
			m.botsWouldMatch = len(withBots)
		}
	}

	m.keepCursorOnVisit()
	return nil
}

// keepCursorOnVisit re-finds the visit the reader was on after the list has
// changed under them.
//
// The fallback matters as much as the rule. Landing on whatever now occupies
// the old row number is exactly the behaviour anchoring by visit exists to
// avoid — hide bots while sitting on a bot visit and you would be handed an
// unrelated human visit chosen by position alone. So a lost anchor goes to the
// top, where a reader looks anyway after changing what they searched for.
func (m *Model) keepCursorOnVisit() {
	if m.cursorVisit != "" {
		for i, s := range m.sessions {
			if s.ID == m.cursorVisit {
				m.cursor = i
				return
			}
		}
	}
	m.cursor, m.offset = 0, 0
	m.rememberCursorVisit()
}

// rememberCursorVisit records which visit the cursor is on, so a later reload
// can put it back.
func (m *Model) rememberCursorVisit() {
	if m.cursor >= 0 && m.cursor < len(m.sessions) {
		m.cursorVisit = m.sessions[m.cursor].ID
		return
	}
	m.cursorVisit = ""
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyPressMsg:
		// While a filter is being typed the keyboard belongs to the text box.
		// Every single-letter binding below is a letter someone might want to
		// search for, so they cannot also be commands.
		if m.filtering {
			return m, m.typeFilter(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			// esc clears the filter and only quits when there is nothing to
			// clear. q and ctrl+c always quit, so this costs no way out.
			if m.filter != "" {
				m.setFilter("")
				break
			}
			return m, tea.Quit

		case "/":
			// Filtering selects visits, and the overview has no visit list to
			// show the result in — pressing / there used to open a filter that
			// narrowed a list off screen while the aggregates beside it stayed
			// put. Take the reader to where the answer is visible.
			m.screen = screenStream
			m.filtering = true

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.rememberCursorVisit()
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
			m.rememberCursorVisit()
		case "g", "home":
			m.cursor = 0
			m.rememberCursorVisit()
		case "G", "end":
			m.cursor = max(0, len(m.sessions)-1)
			m.rememberCursorVisit()
		case "pgup":
			m.cursor = max(0, m.cursor-10)
			m.rememberCursorVisit()
		case "pgdown":
			m.cursor = min(len(m.sessions)-1, m.cursor+10)
			m.rememberCursorVisit()

		case "enter", "space", "right", "l":
			m.toggleExpand()
		case "left", "h":
			if s := m.current(); s != nil {
				delete(m.expanded, s.ID)
			}

		case "b":
			m.showBots = !m.showBots
			// Deliberately does not reset the cursor: the reader did not go
			// looking for anything, the list just grew or shrank around them.
			m.err = m.reload()

		case "tab", "o":
			if m.screen == screenStream {
				m.screen = screenOverview
			} else {
				m.screen = screenStream
			}

		case "r":
			m.rng = m.rng.next()
			m.err = m.reload()

		case "?":
			m.showHelp = !m.showHelp
		}
	}
	return m, nil
}

// typeFilter handles the keyboard while the filter box is open.
func (m *Model) typeFilter(msg tea.KeyPressMsg) tea.Cmd {
	switch key := msg.String(); key {
	case "ctrl+c":
		return tea.Quit

	case "esc":
		m.filtering = false
		m.setFilter("")

	case "enter":
		// Commit: keep what was typed, hand the keyboard back.
		m.filtering = false

	case "backspace":
		if r := []rune(m.filter); len(r) > 0 {
			m.setFilter(string(r[:len(r)-1]))
		}

	// Arrows still navigate, so results can be read without committing first.
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		m.rememberCursorVisit()
	case "down":
		if m.cursor < len(m.sessions)-1 {
			m.cursor++
		}
		m.rememberCursorVisit()

	case "space":
		m.setFilter(m.filter + " ")

	default:
		if r := []rune(key); len(r) == 1 {
			m.setFilter(m.filter + key)
		}
	}
	return nil
}

// setFilter changes the query and re-runs it, keeping the reader on the visit
// they were reading where it survives.
func (m *Model) setFilter(q string) {
	if q == m.filter {
		return
	}
	m.filter = q
	m.err = m.reloadSessions()
}

func (m *Model) current() *store.Session {
	if m.cursor < 0 || m.cursor >= len(m.sessions) {
		return nil
	}
	return &m.sessions[m.cursor]
}

// toggleExpand loads a visit's steps the first time it is opened. A local
// query is fast enough that lazy loading needs no loading state.
func (m *Model) toggleExpand() {
	s := m.current()
	if s == nil {
		return
	}
	if m.expanded[s.ID] {
		delete(m.expanded, s.ID)
		return
	}
	if _, ok := m.hits[s.ID]; !ok {
		hits, err := m.db.SessionHits(context.Background(), s.ID)
		if err != nil {
			m.err = err
			return
		}
		m.hits[s.ID] = hits
	}
	m.expanded[s.ID] = true
}

// View wraps the rendered screen in a tea.View. Bubble Tea v2 made terminal
// modes declarative, so the alt-screen request lives here rather than in a
// program option.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m *Model) render() string {
	if m.err != nil {
		return errStyle.Render("error: "+m.err.Error()) + "\n\npress q to quit\n"
	}
	if m.showHelp {
		return m.helpView()
	}
	if m.screen == screenOverview {
		return m.overviewView()
	}

	var b strings.Builder
	b.WriteString(m.headerView())
	b.WriteString("\n")

	if len(m.sessions) == 0 {
		b.WriteString(m.emptyView())
		return b.String()
	}

	b.WriteString(m.listHeader())
	b.WriteString("\n")

	lines, cursorLine, blockEnd := m.buildLines()

	// Follow the whole selected block, not just its first line. Scrolling to
	// the header alone leaves an expanded journey below the fold — which is
	// the one thing the reader just asked to see. -6 rather than -4: the
	// table header and rule are two more fixed lines above the scroll area.
	viewport := max(3, m.height-6)
	if cursorLine < m.offset {
		m.offset = cursorLine
	}
	if blockEnd >= m.offset+viewport {
		m.offset = blockEnd - viewport + 1
		// A block taller than the screen wins from the top: better to show
		// the visit and its first steps than its last steps alone.
		if m.offset > cursorLine {
			m.offset = cursorLine
		}
	}
	if m.offset > max(0, len(lines)-viewport) {
		m.offset = max(0, len(lines)-viewport)
	}
	if m.offset < 0 {
		m.offset = 0
	}

	end := min(len(lines), m.offset+viewport)
	b.WriteString(strings.Join(lines[m.offset:end], "\n"))
	b.WriteString("\n")
	b.WriteString(m.footerView(len(lines)))
	return b.String()
}

// buildLines renders every row and reports where the selected visit starts and
// ends, so scrolling can keep an expanded block on screen rather than only its
// header line.
func (m *Model) buildLines() (lines []string, cursorLine, blockEnd int) {
	for i, s := range m.sessions {
		isCursor := i == m.cursor
		if isCursor {
			cursorLine = len(lines)
		}
		lines = append(lines, m.sessionLine(s, isCursor))

		if m.expanded[s.ID] {
			for _, h := range m.hits[s.ID] {
				lines = append(lines, m.hitLine(h))
			}
			lines = append(lines, "")
		}
		if isCursor {
			blockEnd = len(lines) - 1
		}
	}
	return lines, cursorLine, blockEnd
}

// listLead is how many characters sit before the first column in every
// list row: two spaces of indent, the ▶/▼ marker, and the gap after it. The
// header row uses the same width of blank space so its text lines up with
// the data underneath, and hitLine's own indent nests one step past it.
const listLead = 4

// listColumns picks a column set for the terminal width. Three tiers, not a
// generic shrink-to-fit: explicit thresholds match how the rest of this
// package handles narrow terminals (see overviewFooter, headerView) rather
// than a clever algorithm nobody can eyeball. Source shrinks first — Till's
// call, after comparing against claws' table (2026-07-26) — because it's
// the one column still useful cut short; time, country and browser identify
// the visit and events is dropped before either of those would be.
func listColumns(width int) []column {
	switch {
	case width >= 70:
		return withSource(width, 36, []column{
			{"time", 12, false}, {"country", 7, false}, {"browser", 8, false},
			{"source", 0, false}, {"pages", 5, true}, {"events", 6, true}, {"dur", 6, true},
		})
	case width >= 50:
		return withSource(width, 24, []column{
			{"time", 9, false}, {"country", 7, false}, {"browser", 8, false},
			{"source", 0, false}, {"pages", 5, true}, {"dur", 6, true},
		})
	default:
		return withSource(width, 15, []column{
			{"time", 5, false}, {"country", 4, false},
			{"source", 0, false}, {"pages", 3, true}, {"dur", 5, true},
		})
	}
}

// withSource gives the source column whatever width is left after every
// other column and gap in cols, floored so it never vanishes and capped so
// it doesn't stretch pointlessly wide on a large terminal.
func withSource(width, cap int, cols []column) []column {
	overhead := listLead + len(cols) - 1 // gaps between columns
	idx := -1
	for i, c := range cols {
		if c.title == "source" {
			idx = i
			continue
		}
		overhead += c.width
	}
	avail := width - overhead
	if avail > cap {
		avail = cap
	}
	if avail < 4 {
		avail = 4
	}
	cols[idx].width = avail
	return cols
}

// listHeader is the table's header row and rule, rendered once above the
// scrolling rows rather than as part of them — it stays on screen while the
// list scrolls, like claws' header does.
func (m *Model) listHeader() string {
	cols := listColumns(m.width)
	prefix := strings.Repeat(" ", listLead)
	return prefix + header.Render(headerLine(cols)) + "\n" +
		prefix + dim.Render(ruleLine(cols))
}

func (m *Model) sessionLine(s store.Session, isCursor bool) string {
	marker := "▶"
	if m.expanded[s.ID] {
		marker = "▼"
	}

	cols := listColumns(m.width)
	values := map[string]string{
		"time":    formatTime(s.FirstSeen),
		"country": nullStr(s.Country),
		"browser": shortBrowser(nullStr(s.Browser)),
		"source":  referrerLabel(s.Referrer),
		"pages":   fmt.Sprint(s.PageCount),
		"events":  fmt.Sprint(s.EventCount),
		"dur":     formatDuration(s.Duration),
	}
	cells := make([]string, len(cols))
	for i, c := range cols {
		cells[i] = values[c.title]
	}
	line := "  " + marker + " " + dataLine(cols, cells)

	// A visit can match on a page its row never shows, so say which one —
	// appended after the table rather than in it, since it names a whole
	// path rather than a value that fits a column.
	if s.Match != "" {
		if avail := m.width - lipgloss.Width(line) - 4; avail > 3 {
			line += "  ← " + truncate(s.Match, avail-4)
		}
	}

	switch {
	case isCursor:
		return selected.Render(line)
	case s.Bot > 0:
		return botStyle.Render(line)
	default:
		return line
	}
}

func (m *Model) hitLine(h store.SessionHit) string {
	label := pathStyle.Render(h.Path)
	if h.IsEvent {
		label = eventStyle.Render(h.Path)
	}
	return "      " + dim.Render(h.CreatedAt.Local().Format("15:04:05")) + "  " + label
}

// headerView drops detail rather than overflowing. A wrapped header destroys
// the layout, and a TUI gets judged in whatever terminal size someone happens
// to have open.
func (m *Model) headerView() string {
	type segment struct {
		text  string
		style lipgloss.Style
	}

	segs := []segment{{"quinto", header}}
	if m.isDemo {
		segs = append(segs, segment{"DEMO DATA", demoTag})
	}
	if m.synced {
		segs = append(segs, segment{"synced " + humaniseAge(time.Since(m.syncedAt)) + " ago", dim})
	} else {
		segs = append(segs, segment{"never synced", dim})
	}
	segs = append(segs, segment{fmt.Sprintf("%d visits", m.totals.Sessions), dim})
	segs = append(segs, segment{fmt.Sprintf("%d pageviews", m.totals.Hits), dim})

	if m.totals.Bots > 0 {
		state := "hidden"
		if m.showBots {
			state = "shown"
		}
		segs = append(segs, segment{fmt.Sprintf("%d bot visits %s", m.totals.Bots, state), dim})
	}

	// Keep the leading segments and stop once the next one would not fit.
	var out []string
	used := 0
	for i, s := range segs {
		cost := len([]rune(s.text))
		if i > 0 {
			cost += 3 // " · "
		}
		if used+cost > m.width {
			break
		}
		used += cost
		out = append(out, s.style.Render(s.text))
	}
	return strings.Join(out, dim.Render(" · "))
}

// footerView shortens its hints on narrow terminals rather than wrapping.
func (m *Model) footerView(total int) string {
	// While typing, the footer is the text box: it has to show what is being
	// searched for, and say that the letter keys are no longer commands.
	if m.filtering {
		box := "/" + m.filter + "▏"
		for _, hints := range []string{
			fmt.Sprintf("  %d matching · enter keep · esc clear · typing, so letters are text", len(m.sessions)),
			fmt.Sprintf("  %d matching · enter keep · esc clear", len(m.sessions)),
			fmt.Sprintf("  %d · enter · esc", len(m.sessions)),
			"",
		} {
			if len([]rune(box+hints)) <= m.width {
				return selected.Render(box) + dim.Render(hints)
			}
		}
		return selected.Render(truncate(box, m.width))
	}

	pos := ""
	if len(m.sessions) > 0 {
		pos = fmt.Sprintf("%d/%d  ", m.cursor+1, len(m.sessions))
	}
	// An active filter is the reason the list is short, so it outranks the
	// hints — a reader who cannot see it is looking at a list that is lying.
	if m.filter != "" {
		pos = fmt.Sprintf("/%s  %d matching  ", m.filter, len(m.sessions))
	}

	hintSets := []string{
		"↑↓ move · enter expand · / filter · tab overview · b bots · ? help · q quit",
		"↑↓ · enter · / filter · tab · b · ? · q",
		"↑↓ · enter · / · b · ? · q",
		"? help · q quit",
		"?",
	}
	if m.filter != "" {
		hintSets = []string{
			"↑↓ move · enter expand · esc clear filter · ? help · q quit",
			"↑↓ · enter · esc clear · ? · q",
			"esc clear · ? · q",
			"esc · q",
			"",
		}
	}
	for _, hints := range hintSets {
		if len([]rune(pos+hints)) <= m.width {
			return dim.Render(pos + hints)
		}
	}
	return dim.Render(truncate(pos, m.width))
}

func (m *Model) emptyView() string {
	// A filter emptying the list is a different situation from having no data,
	// and the advice for one is useless for the other.
	if m.filter != "" {
		msg := fmt.Sprintf("\n  No visits match %q.\n", m.filter)
		if m.botsWouldMatch > 0 {
			msg += "\n" + dim.Render(fmt.Sprintf(
				"  %d hidden bot visits do match — press b to include them.", m.botsWouldMatch)) + "\n"
		}
		return msg + "\n" + dim.Render("  esc clears the filter.") + "\n"
	}

	if m.isDemo {
		return "\n  No demo data yet — run `quinto demo`.\n"
	}
	msg := "\n  Nothing to show yet.\n\n" +
		"  Run `quinto sync` to pull your pageviews,\n" +
		"  or `quinto demo` to see what this looks like with data.\n"
	if !m.showBots && m.totals.Bots > 0 {
		msg += "\n" + dim.Render(fmt.Sprintf("  %d bot visits are hidden — press b to show them.", m.totals.Bots)) + "\n"
	}
	return msg
}

func (m *Model) helpView() string {
	return header.Render("quinto — keys") + "\n\n" +
		"  ↑ / k        previous visit\n" +
		"  ↓ / j        next visit\n" +
		"  enter, space expand or collapse a visit\n" +
		"  ← / →        collapse / expand\n" +
		"  g / G        first / last\n" +
		"  /            filter the visits\n" +
		"  esc          clear the filter, or quit when there is none\n" +
		"  tab / o      switch stream ⇄ overview\n" +
		"  r            cycle the time range\n" +
		"  b            show or hide bot traffic\n" +
		"  ?            this help\n" +
		"  q            quit\n\n" +
		dim.Render("  A filter matches the landing page, referrer, country and\n"+
			"  browser — and every page inside a visit, so searching for\n"+
			"  a page finds the people who reached it second or third.\n"+
			"  Those rows say which page matched, since it is not one the\n"+
			"  row otherwise shows.\n\n"+
			"  The highlight follows the visit you were reading, not the\n"+
			"  row number. If your visit stops matching it goes to the top.\n\n"+
			"  Bot visits are hidden by default but never deleted —\n"+
			"  the exclusion is a filter you can lift, not lost data.\n\n"+
			"  A visit's duration is shown as \"—\" when it cannot be\n"+
			"  measured: one observation tells you someone arrived,\n"+
			"  not when they left.") + "\n\n" +
		dim.Render("  press ? to go back") + "\n"
}

// Run starts the stream view.
func Run(db *store.DB, isDemo bool) error {
	m, err := New(db, isDemo)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m).Run()
	return err
}
