# quinto

**Web analytics in your terminal — and in your agent's.**

`quinto` syncs your pageviews into a local SQLite file, then reads them. The
terminal UI and the SQL interface look at the same file, so the agent already
running in your terminal can interrogate your traffic directly — no API, no
credentials, no rate limit in the way.

<!-- DEMO GIF: generate with `make demo-gif` (requires vhs), then commit docs/demo.gif -->
![quinto](docs/demo.gif)

---

## What it looks like

One row per visit. Expand it to see the path that visitor took through your
site.

```
quinto · synced 4m ago · 309 visits · 707 pageviews · 111 bot visits hidden
  ▶ Thu 14:04  DE · Firefox · reddit.com                    1 page · 1 ev · 13s
  ▼ Thu 14:03  CH · Safari · Hacker News                 7 pages · 2 ev · 8m28s
      14:03:20  /
      14:03:32  Nav · Process
      14:05:18  /challenges
      14:05:34  /
      14:08:02  /work
      14:09:50  /process
      14:09:56  Nav · Challenges
      14:11:21  /challenges
      14:11:48  /process
24/309  ↑↓ move · enter expand · tab overview · b bots · ? help · q quit
```

`tab` switches to the overview.

```
  73 visitors   130 pageviews   40 events   42/73 single-page   24 bots hidden

  pageviews per day · 7 days
  ▆▆▄▆▃█▃▂
  peak 33/day · 8 of 8 days had traffic

  top pages                     referrers                     countries
  /                          53 direct                     39 DE               23
  /work                      19 Google                     11 US               11
  /writing                   17 Hacker News                10 GB               10
```

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

## For agents

This is the part that makes `quinto` different from a prettier dashboard.

The data is a plain SQLite file. Point your agent at the CLI and it can answer
questions about your traffic without an integration:

```sh
quinto schema
quinto query "select path, count(*) c from hits where bot = 0
              group by 1 order by c desc limit 10" --json
```

- `quinto schema` prints the real DDL, so an agent that has never seen the
  project can write a correct query from the CLI alone.
- `--json` gives machine-readable output; without it you get a table.
- The connection is **read-only twice over** — the file opens `mode=ro` and
  `query_only` is set. An agent exploring your data cannot damage it.
- No database client needs to be installed. The binary is the client.

Two things worth telling your agent, both included in `quinto help`:

- Bots are **stored, not filtered**. Add `where bot = 0` for human traffic.
- `sessions.duration_seconds` is **NULL** for visits with a single
  observation, because you cannot see when someone left. `AVG()` skips those,
  which is the point.

## How it works

```
your site ──▶ GoatCounter ──▶ quinto sync ──▶ quinto.db ──▶ TUI
  (3.5 KB)     (storage)        (the only        (SQLite)  └─▶ quinto query
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

- **Bot classification is GoatCounter's**, not ours. It is good, but it is
  theirs, and quinto stores the verdict rather than second-guessing it.
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
