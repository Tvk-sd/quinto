// Package demo generates plausible analytics so quinto has something worth
// looking at without a live account or real visitors.
//
// This exists for three reasons and all three matter: the README's demo GIF
// (real traffic is too thin to sell the tool), the first-run experience (an
// empty screen teaches nobody anything), and design density — a stream view
// built against six real rows looks broken the moment someone with actual
// traffic opens it.
//
// The data is deliberately unflattering. Most visits are single-page, a third
// of traffic is bots, and plenty of sessions bounce in seconds — because that
// is what web analytics actually looks like, and a demo that shows otherwise
// is a sales pitch rather than a preview.
package demo

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/Tvk-sd/quinto/internal/store"
)

// Options controls generation. The zero value is not useful; use Defaults.
type Options struct {
	Seed     uint64
	Days     int // how far back traffic runs
	Sessions int // total sessions across that range, bots included
	EndsAt   time.Time
}

// Defaults produce roughly a month of traffic for a small site — enough for a
// stream view to look inhabited without pretending to be a busy product.
func Defaults() Options {
	return Options{Seed: 20260725, Days: 28, Sessions: 420, EndsAt: time.Now()}
}

type weighted[T any] struct {
	value  T
	weight int
}

func pick[T any](r *rand.Rand, items []weighted[T]) T {
	total := 0
	for _, it := range items {
		total += it.weight
	}
	n := r.IntN(total)
	for _, it := range items {
		if n < it.weight {
			return it.value
		}
		n -= it.weight
	}
	return items[len(items)-1].value
}

// A small site's shape: an entry, a few real pages, and some writing.
var entryPaths = []weighted[string]{
	{"/", 55}, {"/work", 12}, {"/writing", 10},
	{"/writing/shipping-less", 8}, {"/process", 6},
	{"/challenges", 5}, {"/about", 4},
}

// transitions is a crude journey model: where people go next from a page.
// Crude is fine — the point is that paths through the site look like paths,
// not like independent random draws.
var transitions = map[string][]weighted[string]{
	"/":                      {{"/work", 30}, {"/writing", 25}, {"/process", 15}, {"/about", 12}, {"/challenges", 10}, {"/contact", 8}},
	"/work":                  {{"/process", 30}, {"/challenges", 25}, {"/contact", 20}, {"/", 15}, {"/about", 10}},
	"/writing":               {{"/writing/shipping-less", 40}, {"/writing/discovery-notes", 30}, {"/", 20}, {"/about", 10}},
	"/writing/shipping-less": {{"/writing", 40}, {"/writing/discovery-notes", 30}, {"/about", 20}, {"/contact", 10}},
	"/process":               {{"/work", 35}, {"/challenges", 30}, {"/contact", 20}, {"/", 15}},
	"/challenges":            {{"/process", 35}, {"/work", 30}, {"/contact", 20}, {"/", 15}},
	"/about":                 {{"/contact", 40}, {"/work", 30}, {"/writing", 30}},
}

var referrers = []weighted[[2]string]{
	{[2]string{"", ""}, 42}, // direct
	{[2]string{"Google", "h"}, 20},
	{[2]string{"Hacker News", "g"}, 10},
	{[2]string{"LinkedIn", "h"}, 9},
	{[2]string{"GitHub", "h"}, 6},
	{[2]string{"Bluesky", "h"}, 5},
	{[2]string{"reddit.com", "h"}, 4},
	{[2]string{"newsletter", "c"}, 4},
}

var countries = []weighted[string]{
	{"DE", 30}, {"US", 18}, {"GB", 8}, {"NL", 6}, {"AT", 5}, {"CH", 5},
	{"FR", 5}, {"ES", 4}, {"CA", 4}, {"SE", 3}, {"PL", 3}, {"IN", 3},
	{"BR", 2}, {"AU", 2}, {"JP", 2},
}

var clients = []weighted[[2]string]{
	{[2]string{"Chrome 126", "macOS 14.5"}, 18},
	{[2]string{"Chrome 126", "Windows 11"}, 16},
	{[2]string{"Safari 17.5", "macOS 14.5"}, 14},
	{[2]string{"Safari 17.5", "iOS 17.5"}, 13},
	{[2]string{"Chrome 126", "Android 14"}, 11},
	{[2]string{"Firefox 127", "Windows 11"}, 8},
	{[2]string{"Firefox 127", "Linux"}, 6},
	{[2]string{"Edge 126", "Windows 11"}, 6},
	{[2]string{"Safari 17.5", "iPadOS 17.5"}, 4},
	{[2]string{"Chrome 126", "Linux"}, 4},
}

var botClients = []weighted[[2]string]{
	{[2]string{"", ""}, 40},
	{[2]string{"Chrome 108", "Linux"}, 25},
	{[2]string{"Firefox 102", "Linux"}, 20},
	{[2]string{"Chrome 90", "Windows 10"}, 15},
}

var screens = []weighted[string]{
	{"1920,1080,1", 24}, {"2560,1440,2", 18}, {"1512,982,2", 14},
	{"1440,900,2", 12}, {"390,844,3", 12}, {"430,932,3", 8},
	{"1366,768,1", 7}, {"820,1180,2", 5},
}

// events fire alongside pageviews on the pages that have them.
var eventsByPath = map[string][]string{
	"/":                      {"Nav · Work", "Nav · Process"},
	"/work":                  {"CTA · case_study"},
	"/process":               {"Nav · Challenges"},
	"/challenges":            {"Nav · Process"},
	"/contact":               {"CTA · email_click", "Outbound · linkedin"},
	"/writing":               {"Nav · post"},
	"/writing/shipping-less": {"Outbound · newsletter"},
}

// hourWeights gives traffic a day/night curve. Flat traffic is the signature
// of bots, so a demo without a curve looks fake to anyone who reads analytics.
var hourWeights = [24]int{
	2, 1, 1, 1, 1, 2, 4, 8, 14, 20, 24, 26,
	24, 22, 24, 26, 24, 20, 18, 16, 14, 10, 6, 3,
}

// Generate populates db with synthetic traffic. It is deterministic for a
// given seed apart from EndsAt, which is anchored to the present so the demo
// never looks like a stale archive.
func Generate(ctx context.Context, db *store.DB, opt Options) (sessions, hits int, err error) {
	r := rand.New(rand.NewPCG(opt.Seed, opt.Seed^0x9e3779b9))

	var all []store.Hit
	for i := 0; i < opt.Sessions; i++ {
		start := randomMoment(r, opt)
		// Roughly a third of everything hitting a small site is automated.
		if r.IntN(100) < 32 {
			all = append(all, botSession(r, start, i)...)
			continue
		}
		all = append(all, humanSession(r, start, i)...)
		sessions++
	}
	sessions = opt.Sessions

	inserted, err := db.InsertHits(ctx, all)
	if err != nil {
		return 0, 0, err
	}
	return sessions, inserted, nil
}

// randomMoment picks a timestamp within the window, biased by hour so the
// traffic curve looks like people rather than cron jobs.
func randomMoment(r *rand.Rand, opt Options) time.Time {
	day := r.IntN(opt.Days)
	hour := pickHour(r)
	t := opt.EndsAt.Add(-time.Duration(day) * 24 * time.Hour)
	t = time.Date(t.Year(), t.Month(), t.Day(), hour, r.IntN(60), r.IntN(60), 0, time.UTC)

	// Weekends are quieter for a professional site; drop some of them.
	if wd := t.Weekday(); (wd == time.Saturday || wd == time.Sunday) && r.IntN(100) < 45 {
		t = t.Add(-48 * time.Hour)
	}
	return t
}

func pickHour(r *rand.Rand) int {
	total := 0
	for _, w := range hourWeights {
		total += w
	}
	n := r.IntN(total)
	for h, w := range hourWeights {
		if n < w {
			return h
		}
		n -= w
	}
	return 12
}

// humanSession walks a plausible path through the site. Most people read one
// page and leave; a few go deep.
func humanSession(r *rand.Rand, start time.Time, id int) []store.Hit {
	pages := 1
	switch n := r.IntN(100); {
	case n < 58: // single page — the common case, and the honest one
		pages = 1
	case n < 76:
		pages = 2
	case n < 87:
		pages = 3
	case n < 94:
		pages = 4
	default:
		pages = 5 + r.IntN(4)
	}

	session := fmt.Sprintf("demo-h-%04d", id)
	ref := pick(r, referrers)
	client := pick(r, clients)
	country := pick(r, countries)
	screen := pick(r, screens)

	path := pick(r, entryPaths)
	at := start

	var out []store.Hit
	for i := 0; i < pages; i++ {
		h := store.Hit{
			Key:        fmt.Sprintf("demo-%s-%02d", session, i),
			Session:    session,
			Path:       path,
			Title:      titleFor(path),
			Browser:    client[0],
			System:     client[1],
			Country:    country,
			ScreenSize: screen,
			FirstVisit: i == 0,
			CreatedAt:  at.UTC().Format(time.RFC3339),
		}
		if i == 0 {
			h.Referrer, h.ReferrerScheme = ref[0], ref[1]
		} else {
			h.ReferrerScheme = "o"
		}
		out = append(out, h)

		// Some pages emit an event moments after the pageview.
		if evs := eventsByPath[path]; len(evs) > 0 && r.IntN(100) < 35 {
			at = at.Add(time.Duration(2+r.IntN(20)) * time.Second)
			out = append(out, store.Hit{
				Key:        fmt.Sprintf("demo-%s-%02d-ev", session, i),
				Session:    session,
				Path:       evs[r.IntN(len(evs))],
				Title:      "",
				IsEvent:    true,
				Browser:    client[0],
				System:     client[1],
				Country:    country,
				ScreenSize: screen,
				CreatedAt:  at.UTC().Format(time.RFC3339),
			})
		}

		next, ok := transitions[path]
		if !ok {
			break
		}
		path = pick(r, next)
		at = at.Add(time.Duration(8+r.IntN(170)) * time.Second)
	}
	return out
}

// botSession is one hit, no referrer, no journey — which is exactly how
// crawlers appear in real data, and why the bot column earns its place.
func botSession(r *rand.Rand, start time.Time, id int) []store.Hit {
	client := pick(r, botClients)
	paths := []string{"/", "/robots.txt", "/sitemap.xml", "/wp-login.php", "/writing", "/.env", "/work"}

	return []store.Hit{{
		Key:        fmt.Sprintf("demo-b-%04d", id),
		Session:    fmt.Sprintf("demo-b-%04d", id),
		Path:       paths[r.IntN(len(paths))],
		Bot:        1 + r.IntN(3),
		Browser:    client[0],
		System:     client[1],
		Country:    pick(r, countries),
		FirstVisit: true,
		// Bots ignore the day/night curve entirely.
		CreatedAt: start.Add(time.Duration(r.IntN(24)) * time.Hour).UTC().Format(time.RFC3339),
	}}
}

func titleFor(path string) string {
	switch path {
	case "/":
		return "Till von Krueger — Product Manager"
	case "/work":
		return "Work"
	case "/process":
		return "Process"
	case "/challenges":
		return "Challenges"
	case "/about":
		return "About"
	case "/contact":
		return "Contact"
	case "/writing":
		return "Writing"
	case "/writing/shipping-less":
		return "Shipping less, on purpose"
	case "/writing/discovery-notes":
		return "Notes on discovery"
	}
	return path
}
