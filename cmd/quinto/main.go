// Command quinto reads web analytics from a local SQLite file.
//
// The same file backs both the terminal UI and the `query` subcommand, so an
// agent running in the same terminal can interrogate the data directly — no
// API, no credentials, no rate limit between it and the answer.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Tvk-sd/quinto/internal/config"
	"github.com/Tvk-sd/quinto/internal/demo"
	"github.com/Tvk-sd/quinto/internal/goatcounter"
	"github.com/Tvk-sd/quinto/internal/store"
	"github.com/Tvk-sd/quinto/internal/tui"
)

const usage = `quinto — web analytics in your terminal

USAGE
  quinto                        Open the dashboard
  quinto sites                  List configured sites and their last sync
  quinto list                   Print recent visits as a table
  quinto sync                   Pull new pageviews from GoatCounter
  quinto demo                   Fill a separate database with sample traffic
  quinto query <sql> [--json]   Run SQL against the local database
  quinto schema                 Print the database schema
  quinto path                   Print the database file location

FLAGS
  --demo         Read the demo database instead of your own data
  --db <path>    Use a specific database file
  --site <name>  Use a named site from ~/.config/quinto/config
  --json         Emit JSON instead of a table (for scripts and agents)

FOR AGENTS
  The data is a plain SQLite file. Start with 'quinto schema', then query it.

    quinto schema
    quinto query "select path, count(*) c from hits where bot = 0
                  group by 1 order by c desc limit 10" --json

  Two things worth knowing before you write a query:

    - Bots are stored, not filtered. Add 'where bot = 0' for human traffic.
    - sessions.duration_seconds is NULL for single-page visits, because the
      real duration is unobservable. AVG() skips those, which is intended.

  Queries are read-only; the connection cannot modify or drop anything.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("quinto", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	dbPath := fs.String("db", "", "path to the database file")
	asJSON := fs.Bool("json", false, "emit JSON instead of a table")
	demoMode := fs.Bool("demo", false, "read the demo database")
	site := fs.String("site", "", "use a named site from the config file")

	// Split the subcommand from its flags so `quinto query "..." --json`
	// works in the order people actually type it.
	//
	// Flags that take a value must carry it across the split. Sorting purely
	// by the leading dash left `--db` behind while its path went into the
	// positional list, so `--db /some/file` failed with "flag needs an
	// argument" — a documented flag that never worked in its documented form.
	// --site needs the same treatment, for the same reason.
	takesValue := map[string]bool{"-db": true, "--db": true, "-site": true, "--site": true}

	var positional []string
	var flags []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case !strings.HasPrefix(a, "-"):
			positional = append(positional, a)
		case takesValue[a] && i+1 < len(args):
			flags = append(flags, a, args[i+1])
			i++
		default:
			flags = append(flags, a)
		}
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}

	// An unrecognised --site fails here, before anything else runs — even
	// for commands that never touch the network. Otherwise a typo silently
	// opens (or creates) an empty database nobody configured, for every
	// command that only reads a db file rather than syncing one.
	//
	// The matched entry's IsDefault also decides the database file below: the
	// default is addressable by its derived name, but it still keeps
	// quinto.db rather than getting a name-derived file of its own.
	siteIsDefault := false
	if *site != "" {
		sites, err := config.Sites()
		if err != nil {
			return err
		}
		known := false
		names := make([]string, len(sites))
		for i, s := range sites {
			names[i] = s.Name
			if s.Name == *site {
				known = true
				siteIsDefault = s.IsDefault
			}
		}
		if !known {
			if len(names) == 0 {
				return fmt.Errorf("no such site %q — no sites are configured in ~/.config/quinto/config", *site)
			}
			return fmt.Errorf("no such site %q; configured: %s", *site, strings.Join(names, ", "))
		}
	}

	// Demo data lives in its own file. Keeping it separate — rather than
	// tagging rows inside the shared schema — means demo mode cannot touch
	// real data, and the schema an agent reads stays free of bookkeeping.
	isDemo := *demoMode || (len(positional) > 0 && positional[0] == "demo")

	path := *dbPath
	if path == "" {
		name := "quinto.db"
		switch {
		// Demo wins over --site rather than erroring on the combination:
		// demo data isn't per-site, there's exactly one demo database, and
		// `--demo --site X` has an obvious reading (show me sample data)
		// rather than an ambiguous one.
		case isDemo:
			name = "quinto-demo.db"
		case *site != "" && !siteIsDefault:
			name = *site + ".db"
		}
		p, err := store.DefaultPath(name)
		if err != nil {
			return err
		}
		path = p
	}

	if len(positional) == 0 {
		// A config.Sites() failure here falls through to today's exact
		// runTUI call rather than propagating — the bare invocation never
		// needed config before the picker existed, and it shouldn't start
		// needing it now just to fail on an edge case runTUI itself doesn't
		// care about.
		if sites, err := config.Sites(); err == nil && shouldShowPicker(isDemo, *dbPath, *site, len(sites)) {
			return runPicker()
		}
		return runTUI(path, isDemo, *site)
	}

	switch positional[0] {
	case "sync":
		return runSync(path, *site)
	case "demo":
		return runDemo(path)
	case "list":
		return runList(path, isDemo)
	case "query":
		if len(positional) < 2 {
			return fmt.Errorf("query needs SQL: quinto query \"select * from sessions limit 5\"")
		}
		return runQuery(path, strings.Join(positional[1:], " "), *asJSON)
	case "schema":
		return runSchema(path)
	case "path":
		fmt.Println(path)
		return nil
	case "sites":
		return runSites()
	case "help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q — run `quinto help`", positional[0])
	}
}

func runQuery(path, query string, asJSON bool) error {
	db, err := store.OpenReadOnly(path)
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := db.Query(context.Background(), query)
	if err != nil {
		// SQLite's messages are genuinely useful ("no such column: pth").
		// Surface them plainly rather than wrapping them in Go noise.
		return fmt.Errorf("%s", err)
	}

	if asJSON {
		return writeJSON(res)
	}
	return writeTable(res)
}

// runSync pulls everything GoatCounter has since the stored cursor.
//
// Exports are rate limited to roughly one an hour, so hitting that limit is a
// normal outcome, not a failure: it reports when the next sync is possible and
// exits successfully.
func runSync(path, site string) error {
	cfg, err := config.Load(site)
	if err != nil {
		return err
	}

	db, err := store.Open(path)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	state, err := db.SyncState(ctx)
	if err != nil {
		return err
	}

	var cursor *int64
	if state.Synced && state.LastHitID > 0 {
		cursor = &state.LastHitID
		fmt.Fprintf(os.Stderr, "syncing from hit %d…\n", state.LastHitID)
	} else {
		fmt.Fprintln(os.Stderr, "first sync — fetching everything GoatCounter has…")
	}

	client := goatcounter.New(cfg.Site, cfg.Token)

	job, err := client.CreateExport(ctx, cursor)
	if err != nil {
		var rl *goatcounter.RateLimitError
		if errors.As(err, &rl) {
			fmt.Printf("Nothing to do — GoatCounter allows about one export an hour.\n"+
				"Next sync possible in %s.\n", rl.RetryAfter.Round(time.Minute))
			return nil
		}
		return err
	}

	done, err := client.WaitForExport(ctx, job.ID, 2*time.Second)
	if err != nil {
		return err
	}

	raw, err := client.DownloadExport(ctx, job.ID)
	if err != nil {
		return err
	}

	hits, err := goatcounter.ParseExport(bytes.NewReader(raw))
	if err != nil {
		return err
	}

	inserted, err := db.InsertHits(ctx, toStoreHits(hits))
	if err != nil {
		return err
	}
	if err := db.RecordSync(ctx, done.LastHitID, time.Now(), int64(inserted)); err != nil {
		return err
	}

	switch {
	case len(hits) == 0:
		fmt.Println("No new pageviews since the last sync.")
	case inserted == 0:
		fmt.Printf("%d pageviews fetched, all already stored.\n", len(hits))
	default:
		fmt.Printf("Synced %d new pageviews.\n", inserted)
	}
	return nil
}

// toStoreHits converts between the two packages' hit types. They are
// deliberately separate: the parser must not depend on the store, nor the
// store on the API's shape.
func toStoreHits(in []goatcounter.Hit) []store.Hit {
	out := make([]store.Hit, len(in))
	for i, h := range in {
		out[i] = store.Hit{
			Key: h.Key, Path: h.Path, Title: h.Title, IsEvent: h.IsEvent,
			Browser: h.Browser, System: h.System, Session: h.Session, Bot: h.Bot,
			Referrer: h.Referrer, ReferrerScheme: h.ReferrerScheme,
			ScreenSize: h.ScreenSize, Country: h.Country,
			FirstVisit: h.FirstVisit, CreatedAt: h.CreatedAt,
		}
	}
	return out
}

// runDemo fills the demo database with sample traffic. It writes to a
// different file than sync, so it can never overwrite real data — the
// separation is structural rather than a guard someone has to remember.
//
// It replaces the demo data rather than merging into it. Merging seemed safe
// because hit keys are stable across runs, but the timestamps are anchored to
// the run's wall clock: two runs on different days stitched hits weeks apart
// into single sessions, and a third of the sample ended up showing visits that
// lasted days.
func runDemo(path string) error {
	db, err := store.Open(path)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Reset(ctx); err != nil {
		return err
	}

	opts := demo.Defaults()
	sessions, hits, err := demo.Generate(ctx, db, opts)
	if err != nil {
		return err
	}
	if err := db.RecordSync(ctx, 0, time.Now(), int64(hits)); err != nil {
		return err
	}

	fmt.Printf("Generated %d sessions (%d pageviews) over %d days in %s.\n",
		sessions, hits, opts.Days, path)
	fmt.Println("View it with: quinto --demo")
	return nil
}

// runTUI opens the dashboard: the overview first, tab away to the stream.
func runTUI(path string, isDemo bool, site string) error {
	db, err := store.OpenReadOnly(path)
	if err != nil {
		return err
	}
	defer db.Close()

	return tui.Run(db, isDemo, siteLabel(isDemo, site))
}

// siteLabel decides whether the TUI header needs to say which site it's
// showing. A plain single-site setup renders exactly as before named sites
// existed — the label only appears once which site is on screen could
// actually be ambiguous.
func siteLabel(isDemo bool, site string) string {
	if isDemo {
		return ""
	}
	if site != "" {
		return site
	}
	if sites, err := config.Sites(); err == nil && len(sites) > 1 {
		return sites[0].Name
	}
	return ""
}

// runList prints recent visits as a plain table. The TUI is the main
// interface, but a pipeable list is what you want inside a script — or when
// the terminal isn't interactive.
func runList(path string, isDemo bool) error {
	db, err := store.OpenReadOnly(path)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	res, err := db.Query(ctx, `
		SELECT substr(first_seen, 6, 5) || ' ' || substr(first_seen, 12, 5) AS "when",
		       country,
		       browser,
		       COALESCE(NULLIF(referrer, ''), 'direct') AS referrer,
		       page_count AS pages,
		       COALESCE(duration_seconds || 's', '-') AS duration,
		       COALESCE(entry_path, '?') AS entry
		FROM sessions
		WHERE bot = 0
		ORDER BY first_seen DESC
		LIMIT 25`)
	if err != nil {
		return err
	}

	// Never let demo data pass for real traffic. A tool whose screenshots
	// imply numbers its owner does not have is the one failure mode that
	// would undermine the point of building it.
	if isDemo {
		fmt.Println("DEMO DATA — generated sample traffic, not a real site")
	}

	if state, serr := db.SyncState(ctx); serr == nil && state.Synced {
		age := time.Since(state.LastSyncedAt).Round(time.Minute)
		fmt.Printf("data as of %s (%s ago) · %d pageviews stored\n\n",
			state.LastSyncedAt.Local().Format("15:04"), age, state.SyncedRows)
	} else {
		fmt.Println("never synced — run `quinto sync`")
	}

	return writeTable(res)
}

// runSites lists every configured site and when it was last synced. It never
// creates a database for a site that's never been touched — os.Stat decides
// whether there's anything to open, same as runList's "never synced" case.
func runSites() error {
	sites, err := config.Sites()
	if err != nil {
		return err
	}
	if len(sites) == 0 {
		fmt.Println("No sites configured — set site/token in ~/.config/quinto/config, " +
			"or QUINTO_GOATCOUNTER_SITE and QUINTO_GOATCOUNTER_TOKEN.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tHOST\tLAST SYNCED")
	for _, s := range sites {
		name := s.Name + ".db"
		if s.IsDefault {
			name = "quinto.db"
		}
		dbPath, err := store.DefaultPath(name)
		if err != nil {
			return err
		}

		// A missing file and a broken one are different states — the first is
		// "run quinto sync", the second is a bug worth seeing, not a database
		// that looks identical to one nobody has touched yet.
		lastSynced := "never synced"
		if _, statErr := os.Stat(dbPath); statErr == nil {
			db, openErr := store.OpenReadOnly(dbPath)
			if openErr != nil {
				lastSynced = fmt.Sprintf("unreadable: %s", openErr)
			} else {
				if state, sErr := db.SyncState(context.Background()); sErr != nil {
					lastSynced = fmt.Sprintf("unreadable: %s", sErr)
				} else if state.Synced {
					lastSynced = time.Since(state.LastSyncedAt).Round(time.Minute).String() + " ago"
				}
				db.Close()
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Name, s.Site, lastSynced)
	}
	return w.Flush()
}

// shouldShowPicker decides whether the bare `quinto` invocation opens the
// multi-site start screen or goes straight to the dashboard. Any explicit
// override (--db, --site, demo) bypasses it entirely, and it never appears
// with one or zero sites configured — matching today's exact behaviour for
// that case (ticket 02's Design decision #6).
func shouldShowPicker(isDemo bool, dbPath, site string, configuredSites int) bool {
	return !isDemo && dbPath == "" && site == "" && configuredSites > 1
}

// siteDBPath resolves a chosen site's database file by name, the same way
// the --site flag's path resolution does — used by the picker, which works
// with names the reader picked interactively rather than an already-validated
// flag value.
func siteDBPath(name string) (string, error) {
	sites, err := config.Sites()
	if err != nil {
		return "", err
	}
	for _, s := range sites {
		if s.Name != name {
			continue
		}
		if s.IsDefault {
			return store.DefaultPath("quinto.db")
		}
		return store.DefaultPath(name + ".db")
	}
	return "", fmt.Errorf("no such site %q", name)
}

// siteRows builds the picker's data: every configured site's totals and
// last-sync time, plus a synthesized demo row if quinto-demo.db exists.
// Recomputed on every return to the picker rather than cached once — a
// fresh sync or a dashboard visit is exactly when the freshness column
// needs to be current.
func siteRows() ([]tui.SiteRow, error) {
	sites, err := config.Sites()
	if err != nil {
		return nil, err
	}

	rows := make([]tui.SiteRow, 0, len(sites)+1)
	for _, s := range sites {
		name := s.Name + ".db"
		if s.IsDefault {
			name = "quinto.db"
		}
		path, err := store.DefaultPath(name)
		if err != nil {
			return nil, err
		}
		rows = append(rows, siteRow(s.Name, path))
	}

	if demoPath, err := store.DefaultPath("quinto-demo.db"); err == nil {
		if _, statErr := os.Stat(demoPath); statErr == nil {
			row := siteRow("quinto-demo", demoPath)
			row.IsDemo = true
			row.LastSynced = "demo data"
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// siteRow reads one site's totals and sync state from its database file. A
// missing file is "never synced", not an error — a normal state for a site
// nobody has synced yet. A file that exists but won't open or query reports
// why, the same distinction `quinto sites` already makes, rather than
// looking identical to a site nobody has touched.
func siteRow(name, path string) tui.SiteRow {
	row := tui.SiteRow{Name: name, LastSynced: "never synced"}

	if _, statErr := os.Stat(path); statErr != nil {
		return row
	}
	db, err := store.OpenReadOnly(path)
	if err != nil {
		row.LastSynced = fmt.Sprintf("unreadable: %s", err)
		return row
	}
	defer db.Close()

	totals, err := db.Totals(context.Background())
	if err != nil {
		row.LastSynced = fmt.Sprintf("unreadable: %s", err)
		return row
	}
	row.Visits, row.Pageviews = totals.Sessions, totals.Hits

	if state, err := db.SyncState(context.Background()); err == nil && state.Synced {
		row.LastSynced = time.Since(state.LastSyncedAt).Round(time.Minute).String() + " ago"
	}
	return row
}

// runPicker is the multi-site start screen's event loop. Opening a
// dashboard or running a sync both exit the picker's program; looping back
// here to launch it again is what "going back to the list" actually is —
// there is no back keybinding inside the picker or the dashboard themselves
// (ticket 02's Design decision #6).
func runPicker() error {
	for {
		rows, err := siteRows()
		if err != nil {
			return err
		}

		final, err := tea.NewProgram(tui.NewPicker(rows)).Run()
		if err != nil {
			return err
		}
		result := final.(*tui.Picker).Result()

		switch result.Action {
		case tui.ActionQuit:
			return nil

		case tui.ActionOpen:
			if result.Site == "quinto-demo" {
				path, err := store.DefaultPath("quinto-demo.db")
				if err != nil {
					return err
				}
				if err := runTUI(path, true, ""); err != nil {
					return err
				}
				continue
			}
			path, err := siteDBPath(result.Site)
			if err != nil {
				return err
			}
			if err := runTUI(path, false, result.Site); err != nil {
				return err
			}

		case tui.ActionSync:
			path, err := siteDBPath(result.Site)
			if err != nil {
				return err
			}
			if err := runSync(path, result.Site); err != nil {
				return err
			}
			// Without this, the sync's own status line (or the rate-limit
			// message) flashes by the instant the picker's alt-screen takes
			// over again — the opposite of "reports when the next slot
			// opens" if nobody can actually read it.
			fmt.Print("\npress enter to continue…")
			fmt.Scanln()
		}
	}
}

func runSchema(path string) error {
	db, err := store.OpenReadOnly(path)
	if err != nil {
		return err
	}
	defer db.Close()

	ddl, err := db.Schema(context.Background())
	if err != nil {
		return err
	}
	fmt.Println(ddl)
	return nil
}

// writeJSON emits an array of objects — the shape a consumer expects without
// having to correlate a separate column list.
func writeJSON(res *Result) error {
	out := make([]map[string]any, 0, len(res.Rows))
	for _, row := range res.Rows {
		m := make(map[string]any, len(res.Columns))
		for i, col := range res.Columns {
			m[col] = row[i]
		}
		out = append(out, m)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func writeTable(res *Result) error {
	if len(res.Rows) == 0 {
		fmt.Println("(no rows)")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(res.Columns, "\t"))

	for _, row := range res.Rows {
		cells := make([]string, len(row))
		for i, v := range row {
			cells[i] = format(v)
		}
		fmt.Fprintln(w, strings.Join(cells, "\t"))
	}
	return w.Flush()
}

// format renders a value for the terminal. NULL is printed as NULL rather than
// as an empty cell — the difference between "unknown" and "empty" is load
// bearing in this schema.
func format(v any) string {
	if v == nil {
		return "NULL"
	}
	return fmt.Sprintf("%v", v)
}

// Result aliases the store type so this file reads without a package prefix
// on every use.
type Result = store.Result
