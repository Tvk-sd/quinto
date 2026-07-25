package store

import (
	"context"
	"fmt"
	"time"
)

// Overview is the "how is the site doing" summary. Unlike journeys, these
// numbers stay meaningful when n is small — which is why this screen survived
// the scope cuts that removed funnels and path analysis.
type Overview struct {
	Visitors  int // distinct sessions
	Pageviews int // hits that are not events
	Events    int
	Bots      int

	TopPages     []Count
	TopReferrers []Count
	TopCountries []Count
	Daily        []DayCount

	// Bounces are visits with a single pageview. Reported as a count rather
	// than a rate so a reader can see the denominator.
	SinglePageVisits int
}

type Count struct {
	Label string
	N     int
}

type DayCount struct {
	Day time.Time
	N   int
}

// LoadOverview summarises human traffic since a cutoff. A zero cutoff means
// all of it.
func (db *DB) LoadOverview(ctx context.Context, since time.Time) (Overview, error) {
	var o Overview

	cutoff := "0000-01-01T00:00:00Z"
	if !since.IsZero() {
		cutoff = since.UTC().Format(time.RFC3339)
	}

	err := db.QueryRowContext(ctx, `
		SELECT
			(SELECT count(DISTINCT session) FROM hits WHERE bot = 0 AND created_at >= ?1),
			(SELECT count(*) FROM hits WHERE bot = 0 AND is_event = 0 AND created_at >= ?1),
			(SELECT count(*) FROM hits WHERE bot = 0 AND is_event = 1 AND created_at >= ?1),
			(SELECT count(DISTINCT session) FROM hits WHERE bot > 0 AND created_at >= ?1),
			(SELECT count(*) FROM sessions WHERE bot = 0 AND page_count = 1 AND first_seen >= ?1)`,
		cutoff).Scan(&o.Visitors, &o.Pageviews, &o.Events, &o.Bots, &o.SinglePageVisits)
	if err != nil {
		return o, fmt.Errorf("loading overview totals: %w", err)
	}

	if o.TopPages, err = db.topCounts(ctx, cutoff,
		`SELECT path, count(*) n FROM hits
		 WHERE bot = 0 AND is_event = 0 AND created_at >= ?1
		 GROUP BY 1 ORDER BY n DESC LIMIT 8`); err != nil {
		return o, err
	}

	// An empty referrer is a direct visit; that is a real category, not a gap.
	if o.TopReferrers, err = db.topCounts(ctx, cutoff,
		`SELECT CASE WHEN referrer IS NULL OR referrer = '' THEN 'direct' ELSE referrer END,
		        count(*) n
		 FROM sessions WHERE bot = 0 AND first_seen >= ?1
		 GROUP BY 1 ORDER BY n DESC LIMIT 8`); err != nil {
		return o, err
	}

	if o.TopCountries, err = db.topCounts(ctx, cutoff,
		`SELECT COALESCE(NULLIF(country, ''), '??'), count(*) n
		 FROM sessions WHERE bot = 0 AND first_seen >= ?1
		 GROUP BY 1 ORDER BY n DESC LIMIT 6`); err != nil {
		return o, err
	}

	if o.Daily, err = db.dailySeries(ctx, since); err != nil {
		return o, err
	}
	return o, nil
}

func (db *DB) topCounts(ctx context.Context, cutoff, query string) ([]Count, error) {
	rows, err := db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("loading counts: %w", err)
	}
	defer rows.Close()

	var out []Count
	for rows.Next() {
		var c Count
		if err := rows.Scan(&c.Label, &c.N); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// dailySeries returns one entry per day in the range, including days with no
// traffic. Omitting empty days would draw a chart that implies continuous
// activity — the sparse reality is the point.
func (db *DB) dailySeries(ctx context.Context, since time.Time) ([]DayCount, error) {
	var start time.Time
	if since.IsZero() {
		var first string
		err := db.QueryRowContext(ctx,
			`SELECT COALESCE(MIN(created_at), '') FROM hits WHERE bot = 0`).Scan(&first)
		if err != nil {
			return nil, fmt.Errorf("finding first hit: %w", err)
		}
		if first == "" {
			return nil, nil
		}
		start, _ = time.Parse(time.RFC3339, first)
	} else {
		start = since
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)

	rows, err := db.QueryContext(ctx, `
		SELECT substr(created_at, 1, 10) AS day, count(*)
		FROM hits WHERE bot = 0 AND is_event = 0 AND created_at >= ?
		GROUP BY 1`, start.Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("loading daily series: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var day string
		var n int
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		counts[day] = n
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	var out []DayCount
	for d := start; !d.After(today); d = d.AddDate(0, 0, 1) {
		out = append(out, DayCount{Day: d, N: counts[d.Format("2006-01-02")]})
	}
	return out, nil
}
