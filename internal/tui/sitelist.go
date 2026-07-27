package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// SiteRow is one row on the multi-site start screen — a configured site or
// the synthesized demo entry. Built by the caller (cmd/quinto/main.go): this
// package never imports config or goatcounter (see the package comment in
// tui.go), so it has no way to discover sites or credentials on its own.
type SiteRow struct {
	Name       string
	Visits     int
	Pageviews  int
	LastSynced string // "4m ago", "never synced", "demo data", or "unreadable: ..."
	IsDemo     bool
}

// PickerAction is what the reader chose on the start screen. main.go acts on
// it — opening a dashboard or running a sync — then, for anything but quit,
// re-launches the picker: that relaunch is what "going back to the list"
// actually is, not a keybinding inside this screen.
type PickerAction int

const (
	ActionQuit PickerAction = iota
	ActionOpen
	ActionSync
)

// PickerResult is what the picker program exits with. Site is empty only
// for ActionQuit.
type PickerResult struct {
	Action PickerAction
	Site   string
}

// Picker is the start screen: pick a site, or sync one, without syncing
// touching this package — s/S exit with an ActionSync result for main.go to
// carry out, exactly as opening a dashboard exits with ActionOpen.
type Picker struct {
	rows   []SiteRow
	cursor int
	result PickerResult
}

func NewPicker(rows []SiteRow) *Picker {
	return &Picker{rows: rows}
}

// Result is only meaningful after the program has quit.
func (m *Picker) Result() PickerResult { return m.result }

func (m *Picker) Init() tea.Cmd { return nil }

func (m *Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "q", "ctrl+c", "esc":
		m.result = PickerResult{Action: ActionQuit}
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}

	case "enter":
		m.result = PickerResult{Action: ActionOpen, Site: m.rows[m.cursor].Name}
		return m, tea.Quit

	case "s":
		// Demo isn't a configured site — there's nothing for `quinto sync`
		// to fetch for it. Silently doing nothing is more honest here than
		// attempting a sync that config.Load would just reject.
		if !m.rows[m.cursor].IsDemo {
			m.result = PickerResult{Action: ActionSync, Site: m.rows[m.cursor].Name}
			return m, tea.Quit
		}

	case "S":
		if stalest, ok := stalestRow(m.rows); ok {
			m.result = PickerResult{Action: ActionSync, Site: stalest.Name}
			return m, tea.Quit
		}
	}
	return m, nil
}

// stalestRow is what `S` actually syncs now that ticket 05 found the export
// budget is per account, not per site: at most one site can ever sync per
// press, so `S` picks the one furthest from fresh rather than offering all
// of them. Demo never counts — it isn't a real site to freshen, and rows
// that were never synced or came back unreadable sort as infinitely stale so
// they're picked before anything with a real age.
func stalestRow(rows []SiteRow) (SiteRow, bool) {
	var best SiteRow
	found := false
	for _, r := range rows {
		if r.IsDemo {
			continue
		}
		if !found || staleness(r) > staleness(best) {
			best, found = r, true
		}
	}
	return best, found
}

// staleness orders rows for `S` without parsing "4m ago" back into a
// duration: never-synced and unreadable rows are the most stale, real ages
// fall back to string comparison, which is good enough to break ties among
// the rest without needing a real clock here.
func staleness(r SiteRow) int {
	switch {
	case r.LastSynced == "never synced":
		return 2
	case strings.HasPrefix(r.LastSynced, "unreadable"):
		return 2
	default:
		return 1
	}
}

func (m *Picker) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

func (m *Picker) render() string {
	cols := []column{
		{"SITE", 16, false},
		{"VISITS", 8, true},
		{"PAGEVIEWS", 10, true},
		{"LAST SYNCED", 14, false},
	}
	width := 2 + colsWidth(cols) // 2 for the cursor marker

	var b strings.Builder
	b.WriteString(pickerBanner())
	b.WriteString("\n\n")
	b.WriteString(header.Render("  "+headerLine(cols)) + "\n")
	b.WriteString(dim.Render("  "+ruleLine(cols)) + "\n")

	for i, r := range m.rows {
		marker := "  "
		if i == m.cursor {
			marker = "▸ "
		}
		row := padTo(marker+dataLine(cols, []string{
			r.Name, fmt.Sprint(r.Visits), fmt.Sprint(r.Pageviews), r.LastSynced,
		}), width)
		switch {
		case i == m.cursor:
			row = selected.Render(row)
		case r.IsDemo:
			row = demoTag.Render(row)
		}
		b.WriteString(row + "\n")
	}

	b.WriteString("\n" + dim.Render("enter open · s sync · S sync stalest · q quit"))
	return b.String()
}

// pickerBanner is the side-by-side lockup ticket 02's Design section
// settled on: icon left, wordmark and tagline right, vertically centred
// against the icon's height. Matches proto/start-screen-banner's winning
// layout exactly.
func pickerBanner() string {
	icon := goatIcon()
	const iconWidth = 18
	const textWidth = 30

	text := make([]string, len(icon))
	mid := len(icon) / 2
	for i := range text {
		switch i {
		case mid - 1:
			text[i] = header.Render(padTo("Q U I N T O", textWidth))
		case mid:
			text[i] = dim.Render(padTo("web analytics in your terminal", textWidth))
		default:
			text[i] = strings.Repeat(" ", textWidth)
		}
	}

	var b strings.Builder
	for i := range icon {
		b.WriteString("  " + padVisible(icon[i], iconWidth) + "  " + text[i] + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func colsWidth(cols []column) int {
	w := 0
	for i, c := range cols {
		w += c.width
		if i > 0 {
			w++
		}
	}
	return w
}
