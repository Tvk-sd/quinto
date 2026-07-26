# quinto

**Web analytics in your terminal.**

Analytics tools are built for sites with traffic. If yours gets fifty visits a
week, a dashboard of percentages and trend lines tells you nothing true: the
numbers are too small to mean anything, and the interface is designed to hide
that from you. A 58% bounce rate out of nineteen visits is not a measurement.

**quinto is for small sites, where the only honest thing to show is the visits
themselves.** Somebody came from Hacker News, read three pages over eight
minutes, and left from `/work`. That is a real fact about a real person. It is
also the entire customer journey, at the only sample size you actually have.

It syncs your pageviews into a SQLite file on your machine and reads them from
there. The terminal UI and the SQL interface read the same file, so anything
that can open a file can read your traffic.

![quinto](docs/demo.gif)

<!--
  Re-record with `make demo-gif` (needs: brew install vhs) whenever the UI
  changes — the tape walks a fixed path through the app, so a change to which
  screen opens first, or to what the footer says, silently makes this GIF a
  picture of software that no longer exists.
-->

## What it looks like

It opens on the overview: what happened, before who came.

```
quinto · DEMO DATA · synced moments ago · 286 visits · 647 pageviews · 134 bot visits hidden

  66 visitors   109 pageviews   27 events   36/66 single-page   41 bots hidden

  pageviews per day · 7 days
  ▅▃█▄█▇▂▂
  peak 26/day · 8 of 8 days had traffic

  PAGE                        N REFERRER                    N COUNTRY                     N
  ───────────────────────────── ───────────────────────────── ─────────────────────────────
  /                          35 direct                     31 DE                         22
  /work                      18 Google                     16 US                         12
  /writing                   15 LinkedIn                    7 GB                          8
  /process                    9 GitHub                      5 NL                          7

  tab stream · / filter · r range · b bots · ? help · q quit
```

`tab` switches to the stream — one row per visit, expandable to the path that
visitor took through your site.

```
quinto · DEMO DATA · synced moments ago · 286 visits · 647 pageviews · 134 bot visits hidden
    TIME         COUNTRY BROWSER  SOURCE                               PAGES EVENTS    DUR
    ──────────────────────────────────────────────────────────────────────────────────────
  ▶ Tue 16:34    DE      Chrome   GitHub                                   2      1  2m55s
  ▶ Tue 15:08    GB      Chrome   direct                                   1      0      —
  ▶ Tue 14:16    DE      Chrome   direct                                   2      1  3m10s
  ▼ Tue 12:56    NL      Chrome   direct                                   5      1  8m23s
      12:56:51  /challenges
      12:56:56  Nav · Process
      12:59:09  /work
      13:01:54  /
      13:03:25  /writing
      13:05:14  /writing/discovery-notes

  ▶ Tue 11:02    ES      Safari   direct                                   1      0      —
56/286  ↑↓ move · enter expand · / filter · tab overview · b bots · ? help · q quit
```

`/` filters. It matches the landing page, referrer, country and browser — and
every page **inside** a visit, so searching for a page finds the people who
reached it second or third, not only those who arrived on it.

```
quinto · DEMO DATA · synced moments ago · 286 visits · 647 pageviews · 134 bot visits hidden
    TIME         COUNTRY BROWSER  SOURCE                               PAGES EVENTS    DUR
    ──────────────────────────────────────────────────────────────────────────────────────
  ▶ Fri 16:05    JP      Edge     Google                                   2      0    38s
  ▶ Fri 13:34    IN      Chrome   Google                                   4      1  3m48s
  ▶ Fri 10:09    DE      Safari   newsletter                               2      0  1m13s
  ▶ Wed 02:00    DE      Chrome   direct                                   2      1  1m23s
  ▶ 19 Jul 19:15 NL      Chrome   GitHub                                   5      3  6m09s
/contact▏  32 matching · enter keep · esc clear · typing, so letters are text
```

In the sample data nobody lands on `/contact` — 32 visits reach it anyway.
The filter runs as a single SQL statement, so it is also a query you can hand
to an agent.

## Install

Single static binary, no runtime dependencies. macOS and Linux, amd64 and
arm64.

```sh
go install github.com/Tvk-sd/quinto/cmd/quinto@latest
```

Or grab a binary from [Releases](../../releases) and put it on your `PATH`.

## Try it in thirty seconds

You do not need an analytics account to see what this is:

```sh
quinto demo      # generate sample traffic in a separate database
quinto --demo    # look at it
```

The sample data is deliberately unflattering — a third of it is bots, most
visits are single-page, and the traffic curve has real nights in it. A demo
that shows a thriving site would tell you nothing about how the tool behaves
on yours.

Every screen in demo mode is labelled `DEMO DATA`.

## Your own data

`quinto` reads from [GoatCounter](https://www.goatcounter.com), which is open
source, privacy-first, needs no cookie banner, and is free for personal and
small-business use.

<details>
<summary><strong>Why GoatCounter and not Plausible, Umami or Cloudflare?</strong></summary>

Because **raw per-visit rows are the thing analytics vendors charge for**, and
quinto cannot work without them. Aggregates are cheap to serve and get given
away; individual rows are the product. Every candidate was checked on that one
axis first:

| | free | hosted | per-visit rows |
|---|---|---|---|
| **GoatCounter** | ✅ | ✅ | ✅ export API with session IDs and a bot flag |
| PostHog Cloud | ✅ | ✅ | ✅ but a heavy script and a consent banner, to render a list of visits |
| **Plausible** | ❌ no free tier | ✅ | ❌ Stats API is a *Business-plan* feature, and returns aggregates even then |
| Fathom, Simple Analytics | ❌ | ✅ | ❌ aggregated |
| **Umami Cloud** | ❌ API is €20/mo | ✅ | ✅ |
| Cloudflare | ✅ | ✅ | ❌ raw logs are Enterprise-only |

Plausible is the better *product* — nicer dashboard, more polish. It is
unusable here at **any** price, because there is no per-visit row to fetch.
Price is not why it loses; shape is.

GoatCounter is not the prettiest of these. It is the one whose data model
fits — and its export happens to carry exactly the two fields this tool needs
most: a session identifier, so visits can be grouped into journeys, and a bot
classification, so the crawler traffic that dominates small sites can be set
aside without being destroyed.

</details>

**1. Create a GoatCounter site** and add the tracking snippet to your pages.

**2. Turn on individual pageviews.** *Settings → Tracking → Individual
pageviews.*

> This one setting has three traps, and all three cost us time:
> it is **off by default**; the toggle can look enabled **without saving**, so
> reload the page and check; and it is **not retroactive** — only hits recorded
> after it is on ever appear. A dashboard full of pageviews is no evidence it
> is working.

**3. Create an API token** at `https://<your-site>.goatcounter.com/user/api`.

> Tick **Export** and nothing else, and scope it to a single site rather than
> "all sites, including those created in the future". `quinto` only ever calls
> three export endpoints. It does not need *Read statistics* — every aggregate
> is computed locally. It must never hold *Record pageviews*: a leaked
> read-only token costs you a copy of data you already own, but a token with
> write scope costs you your data's integrity.

**4. Point quinto at it.** Either `~/.config/quinto/config`:

```ini
site  = yoursite.goatcounter.com
token = your-export-token
```

or the environment, which takes precedence:

```sh
export QUINTO_GOATCOUNTER_SITE=yoursite.goatcounter.com
export QUINTO_GOATCOUNTER_TOKEN=...
```

**5. Sync and look.**

```sh
quinto sync
quinto
```

> GoatCounter allows roughly **one export per hour**. `quinto sync` is a
> deliberate, occasional action rather than something to run in a loop. When
> the limit is hit it tells you when the next sync is possible and exits
> cleanly — it is a normal state, not an error. Every screen states how old its
> data is, because at that cadence showing stale numbers as current would be
> the interface lying to you.

## Asking it things the screens don't answer

Any dashboard can only answer the questions someone built a screen for. *"Do
the people who reach my contact page come from the newsletter or from
Google?"* If that view doesn't exist, you're stuck exporting a CSV.

quinto's screens have the same limit. But underneath, **the data is a SQLite
file, and there's a subcommand that queries it.** So when the screen runs out,
you don't leave the terminal.

```sh
quinto schema      # the real DDL — two tables, nothing hidden
quinto query "select coalesce(nullif(s.referrer,''),'direct') as referrer, count(*) n
              from sessions s
              join hits h on h.session = s.session and h.path = '/contact'
              where s.bot = 0 group by 1 order by n desc"
```

```
referrer     n
direct       15
Google       5
Hacker News  4
GitHub       3
LinkedIn     2
```

The same subcommand works for an agent as well. **A question costs a single
shell command** and that's it. No MCP server to run, no API client, no auth
handshake, no pagination. If your agent can execute `quinto query`, the
integration is already finished.

That cheapness is the point:

- **`quinto schema` is one call and then it knows everything.** No guessing at
  column names, no burning turns on trial and error before the first useful
  answer.
- **Every follow-up is another cheap call.** Ask three questions to narrow
  something down and it costs three commands against a local file — not three
  round trips through someone's rate-limited API.
- **Nothing is behind credentials.** Your GoatCounter token lives in `sync`
  alone. The query path never touches the network, so an agent reading your
  analytics is never an agent holding your keys.
- **It cannot break anything.** The file opens `mode=ro` *and* sets
  `query_only` — read-only twice over. You can let it explore freely.
- **`--json` when a machine is reading**, a table when you are.
- **No database client needed** on either side. The binary is the client.

Two things worth putting in your agent's instructions, both also in
`quinto help`:

- Bots are **stored, not filtered**. Add `where bot = 0` for human traffic.
- `sessions.duration_seconds` is **NULL** for visits with a single
  observation, because you cannot see when someone left. `AVG()` skips those,
  which is the point.

## How it works

```
your site ──▶ GoatCounter ──▶ quinto sync ──▶ quinto.db ──▶ TUI
  (3.5 KB)     (storage)        (the only        (SQLite)  ├─▶ quinto query
                                network step)               └─▶ your agent
```

The interface never talks to the network. That is what makes it instant,
usable offline, and readable by anything that can open a file.

## Design decisions

The tool is built for sites with modest traffic, which is most sites. That
constraint drives everything:

**Streams, not aggregates.** Funnels and path analysis need volume most sites
do not have. A list of visits is honest at any number. The expandable journey
is a customer journey map in the only form that isn't a lie at n=1.

**Unknown is not zero.** A single-observation visit shows `—`, never `0s`. You
know someone arrived; you do not know when they left. The same rule keeps
averages meaningful, since `AVG` skips NULL rather than being dragged toward
zero by every unmeasurable visit.

**Unknown is not "direct" either.** A referrer we never synced shows `?`. Real
direct traffic shows `direct`. Merging them would invent data.

**Empty days draw nothing.** The traffic chart leaves a gap for a day with no
visits instead of drawing a baseline block, and says *"too few days to call it
a trend"* rather than presenting noise as a line.

**Bounces are a count, not a rate.** `42/73`, not `58%`. At these volumes a
percentage implies precision the sample cannot support.

**Bots are hidden, never deleted.** They are excluded by default and the count
is always shown, so the exclusion is a filter you can lift rather than data
someone threw away for you.

## Limitations

- **Bot classification is GoatCounter's**, not ours. It is good, and it is not
  ours to improve — quinto stores their verdict rather than second-guessing it.
- **One sync per hour**, imposed upstream.
- **Sessions rotate daily by design.** GoatCounter's session identifier is a
  salted hash that changes every day so visitors cannot be tracked across
  days — which is why it needs no consent banner. A visit spanning midnight
  therefore appears as two visits. quinto does not stitch them: that would
  mean guessing two hashes are the same person, precisely the inference the
  design refuses to make.
- **Single-page apps** need an explicit `goatcounter.count()` on route change;
  GoatCounter has no automatic hook for it.
- GoatCounter's hosted service is donation-supported and free "for reasonable
  public usage" — generous, but not a contract.

## Development

```sh
make test     # unit tests
make build    # local binary
make release  # static binaries for macOS and Linux, amd64 and arm64
```

The GoatCounter client is tested against a **real captured export**
(`internal/goatcounter/testdata/`), not a hand-written approximation. That
export contradicted the published docs in four ways, each of which would have
produced a parser that passed its own tests and failed on contact with the API.

## Licence

MIT — see [LICENSE](LICENSE).
