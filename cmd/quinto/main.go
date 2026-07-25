// Command quinto reads web analytics from a local SQLite file.
//
// The same file backs both the terminal UI and the `query` subcommand, so an
// agent running in the same terminal can interrogate the data directly — no
// API, no credentials, no rate limit between it and the answer.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/Tvk-sd/quinto/internal/store"
)

const usage = `quinto — web analytics in your terminal

USAGE
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

	if len(positional) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return nil
	}

	path := *dbPath
	if path == "" {
		p, err := store.DefaultPath("quinto.db")
		if err != nil {
			return err
		}
		path = p
	}

	switch positional[0] {
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
