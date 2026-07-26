package store

import "strings"

// SessionFilter narrows the visit list. The zero value matches every visit
// except bots.
//
// Matching is deliberately substring, not fuzzy. The genre answer for fuzzy is
// sahilm/fuzzy, and adopting it would be a dependency decision rather than a
// behavioural one — nothing here changes if the matcher is swapped later.
type SessionFilter struct {
	Query       string
	IncludeBots bool
}

// Active reports whether a text query is narrowing the list. The bots toggle
// is not a filter in this sense — it is on by default and has its own control.
func (f SessionFilter) Active() bool { return f.Query != "" }

// SQL renders the filter as one statement, with its arguments.
//
// This is the whole of "the same file is queryable by agents": the TUI does not
// filter in Go over something it loaded, it runs a query — so the filter a
// reader sees on screen is a query they could have typed themselves. One
// builder serves both, which makes that parity structural rather than a claim
// that has to be maintained by hand.
//
// The match_reason column is the path of the first page *inside* a visit that
// matched, and empty when the match is already visible on the row. A visit can
// match on a page its row never shows, so a filter that could not explain
// itself would look broken.
func (f SessionFilter) SQL(limit int) (string, []any) {
	const cols = `session, first_seen, last_seen, page_count, event_count, bot,
		       entry_path, referrer, country, browser, system, duration_seconds`

	if !f.Active() {
		where := "WHERE bot = 0"
		if f.IncludeBots {
			where = ""
		}
		return `
		SELECT ` + cols + `, '' AS match_reason
		FROM sessions ` + where + `
		ORDER BY first_seen DESC
		LIMIT ?`, []any{limit}
	}

	like := "%" + escapeLike(strings.ToLower(f.Query)) + "%"

	// Four fields the row already displays, plus the pages inside the visit.
	const onRow = `lower(entry_path) LIKE ? ESCAPE '\'
		    OR lower(referrer)   LIKE ? ESCAPE '\'
		    OR lower(country)    LIKE ? ESCAPE '\'
		    OR lower(browser)    LIKE ? ESCAPE '\'`

	const insideVisit = `SELECT 1 FROM hits h
		         WHERE h.session = sessions.session
		           AND lower(h.path) LIKE ? ESCAPE '\'`

	const firstMatchingStep = `SELECT h.path FROM hits h
		         WHERE h.session = sessions.session
		           AND lower(h.path) LIKE ? ESCAPE '\'
		         ORDER BY h.created_at LIMIT 1`

	bots := "bot = 0 AND"
	if f.IncludeBots {
		bots = ""
	}

	// Argument order follows the order the placeholders appear in the text:
	// the CASE in the select list is read before the WHERE clause.
	args := []any{like, like, like, like, like, like, like, like, like, like, limit}

	return `
		SELECT ` + cols + `,
		       CASE WHEN ` + onRow + `
		            THEN ''
		            ELSE coalesce((` + firstMatchingStep + `), '')
		       END AS match_reason
		FROM sessions
		WHERE ` + bots + ` (` + onRow + `
		    OR EXISTS (` + insideVisit + `))
		ORDER BY first_seen DESC
		LIMIT ?`, args
}

// escapeLike neutralises the wildcards a reader may type without meaning them.
// Without it, typing "%" matches everything and "_" matches anything — which
// looks like the filter is broken rather than like it is working.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}
