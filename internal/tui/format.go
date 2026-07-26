package tui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// formatDuration renders a visit's length. An unmeasurable visit shows "—",
// never "0s": one observation tells you someone arrived, not when they left.
// Printing zero would assert something the data does not say.
func formatDuration(d sql.NullInt64) string {
	if !d.Valid {
		return "—"
	}
	secs := d.Int64
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm%02ds", secs/60, secs%60)
	default:
		return fmt.Sprintf("%dh%02dm", secs/3600, (secs%3600)/60)
	}
}

// formatTime shows the clock for today's visits and adds a date for older
// ones — without it, a list spanning several days reads as out of order.
func formatTime(t time.Time) string {
	local := t.Local()
	now := time.Now()

	if local.YearDay() == now.YearDay() && local.Year() == now.Year() {
		return local.Format("15:04")
	}
	if now.Sub(local) < 7*24*time.Hour {
		return local.Format("Mon 15:04")
	}
	return local.Format("02 Jan 15:04")
}

func humaniseAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "moments"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// referrerLabel keeps "direct" distinct from "unknown". A NULL referrer means
// the entry hit was never synced, which is not the same as someone typing the
// URL in — and the view should not quietly merge the two.
func referrerLabel(r sql.NullString) string {
	if !r.Valid {
		return "?"
	}
	if strings.TrimSpace(r.String) == "" {
		return "direct"
	}
	return r.String
}

// shortBrowser drops version numbers, which cost width and tell a reader
// nothing they act on.
func shortBrowser(b string) string {
	if b == "" {
		return ""
	}
	if i := strings.IndexByte(b, ' '); i > 0 {
		return b[:i]
	}
	return b
}

func nullStr(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func truncate(s string, width int) string {
	if width <= 1 || len([]rune(s)) <= width {
		return s
	}
	return string([]rune(s)[:width-1]) + "…"
}
