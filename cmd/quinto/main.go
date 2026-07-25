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

	"github.com/Tvk-sd/quinto/internal/config"
	"github.com/Tvk-sd/quinto/internal/goatcounter"
	"github.com/Tvk-sd/quinto/internal/store"
)

const usage = `quinto — web analytics in your terminal

USAGE
  quinto                        Show recent visits
  quinto sync                   Pull new pageviews from GoatCounter
  quinto query <sql> [--json]   Run SQL against the local database
  quinto schema                 Print the database schema
  quinto path                   Print the database file location

FLAGS
  --db <path>   Use a specific database file
  --json        Emit JSON instead of a table (for scripts and agents)

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

	// Split the subcommand from its flags so `quinto query "..." --json`
	// works in the order people actually type it.
	var positional []string
	var flags []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
		} else {
			positional = append(positional, a)
		}
	}
	if err := fs.Parse(flags); err != nil {
		return err
	}

	path := *dbPath
	if path == "" {
		p, err := store.DefaultPath("quinto.db")
		if err != nil {
			return err
		}
		path = p
	}

	if len(positional) == 0 {
		return runRecent(path)
	}

	switch positional[0] {
	case "sync":
		return runSync(path)
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
func runSync(path string) error {
	cfg, err := config.Load()
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

// runRecent lists the most recent visits, humans only. It always states how
// old the data is: with an hourly sync ceiling, presenting stale numbers as
// current would be the interface lying.
func runRecent(path string) error {
	db, err := store.OpenReadOnly(path)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	res, err := db.Query(ctx, `
		SELECT substr(first_seen, 12, 5) AS time,
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

	if state, serr := db.SyncState(ctx); serr == nil && state.Synced {
		age := time.Since(state.LastSyncedAt).Round(time.Minute)
		fmt.Printf("data as of %s (%s ago) · %d pageviews stored\n\n",
			state.LastSyncedAt.Local().Format("15:04"), age, state.SyncedRows)
	} else {
		fmt.Println("never synced — run `quinto sync`")
	}

	return writeTable(res)
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
