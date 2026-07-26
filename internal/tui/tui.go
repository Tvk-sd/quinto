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
	dim = lipgloss.NewStyle().Foreground(adaptive("245", "241"))

	header = lipgloss.NewStyle().
		Foreground(adaptive("236", "252")).
		Bold(true)

	demoTag = lipgloss.NewStyle().
		Foreground(adaptive("130", "214")).
		Bold(true)

	selected = lipgloss.NewStyle().
			Foreground(adaptive("27", "117")).
			Bold(true)

	pathStyle = lipgloss.NewStyle().
			Foreground(adaptive("22", "114"))

	eventStyle = lipgloss.NewStyle().
			Foreground(adaptive("97", "141"))

	botStyle = lipgloss.NewStyle().
			Foreground(adaptive("244", "240")).
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
	ctx := context.Background()

	sessions, err := m.db.RecentSessions(ctx, sessionLimit, m.showBots)
	if err != nil {
		return err
	}
	m.sessions = sessions

	if m.totals, err = m.db.Totals(ctx); err != nil {
		return err
	}
	if state, err := m.db.SyncState(ctx); err == nil && state.Synced {
		m.syncedAt, m.synced = state.LastSyncedAt, true
	}

	if m.overview, err = m.db.LoadOverview(ctx, m.rng.since()); err != nil {
		return err
	}

	if m.cursor >= len(m.sessions) {
		m.cursor = max(0, len(m.sessions)-1)
	}
	return nil
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			m.cursor = max(0, len(m.sessions)-1)
		case "pgup":
			m.cursor = max(0, m.cursor-10)
		case "pgdown":
			m.cursor = min(len(m.sessions)-1, m.cursor+10)

		case "enter", "space", "right", "l":
			m.toggleExpand()
		case "left", "h":
			if s := m.current(); s != nil {
				delete(m.expanded, s.ID)
			}

		case "b":
			m.showBots = !m.showBots
			m.cursor, m.offset = 0, 0
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

	lines, cursorLine, blockEnd := m.buildLines()

	// Follow the whole selected block, not just its first line. Scrolling to
	// the header alone leaves an expanded journey below the fold — which is
	// the one thing the reader just asked to see.
	viewport := max(3, m.height-4)
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

func (m *Model) sessionLine(s store.Session, isCursor bool) string {
	marker := "▶"
	if m.expanded[s.ID] {
		marker = "▼"
	}

	when := formatTime(s.FirstSeen)
	who := strings.Join(nonEmpty(
		nullStr(s.Country),
		shortBrowser(nullStr(s.Browser)),
		referrerLabel(s.Referrer),
	), " · ")

	pages := fmt.Sprintf("%d page", s.PageCount)
	if s.PageCount != 1 {
		pages += "s"
	}
	if s.EventCount > 0 {
		pages += fmt.Sprintf(" · %d ev", s.EventCount)
	}

	right := pages + " · " + formatDuration(s.Duration)
	left := fmt.Sprintf("%s %s  %s", marker, when, who)

	// Pad so the right-hand column lines up, but never wrap the terminal.
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
		if trim := m.width - lipgloss.Width(right) - 4; trim > 10 && lipgloss.Width(left) > trim {
			left = truncate(left, trim)
		}
	}
	line := "  " + left + strings.Repeat(" ", gap) + right

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
	pos := ""
	if len(m.sessions) > 0 {
		pos = fmt.Sprintf("%d/%d  ", m.cursor+1, len(m.sessions))
	}

	for _, hints := range []string{
		"↑↓ move · enter expand · tab overview · b bots · ? help · q quit",
		"↑↓ · enter · tab · b · ? · q",
		"↑↓ · enter · b bots · ? · q",
		"? help · q quit",
		"?",
	} {
		if len([]rune(pos+hints)) <= m.width {
			return dim.Render(pos + hints)
		}
	}
	return dim.Render(truncate(pos, m.width))
}

func (m *Model) emptyView() string {
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
		"  tab / o      switch stream ⇄ overview\n" +
		"  r            cycle the time range\n" +
		"  b            show or hide bot traffic\n" +
		"  ?            this help\n" +
		"  q            quit\n\n" +
		dim.Render("  Bot visits are hidden by default but never deleted —\n"+
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
