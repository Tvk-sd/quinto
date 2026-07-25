# quinto — Plan

*From "quinto sabor" — the fifth taste. Umami is the fifth taste, and Umami is the data source.*

**Status:** Portfolio project. Validation phase cut. Ready to scope the build.
**Success criterion:** Listed on [awesome-tuis → Dashboards](https://github.com/rothgar/awesome-tuis#dashboards)
**Last updated:** 2026-07-24

---

## What this is

A terminal-native web analytics dashboard for developers who already run agents in their terminal.

**It is a portfolio project, not a product.** Decided 2026-07-24 after establishing that no dogfooding data exists and the target segment is unvalidated. This is an honest reclassification, not a downgrade — it removes weeks of validation work that would not have changed the build.

Jobs (Till's own framing):

- **Functional** — easy-to-read, *local* analytics
- **Emotional** — I can run it from my terminal
- **Social** — a specific aesthetic, for people already running agents in the terminal

---

## Definition of done

**Primary (verifiable):** merged into awesome-tuis → Dashboards.

**Real bar (what actually gets it noticed — the listing follows this, not the reverse):**

- [ ] A demo GIF in the README that works without explanation. This does more for a TUI than any feature.
- [ ] **A seeded demo dataset (`--demo` mode or checked-in fixture).** First-class deliverable, not an afterthought — see "The GIF problem" below.
- [ ] One hook that isn't "another pretty dashboard." Current candidate: the local data file is agent-queryable.
- [ ] Runs on someone else's machine in under a minute — single binary or one install command.
- [ ] A name worth typing.

### The GIF problem

Real traffic is single-digit humans and probably bot-dominated. A stream view showing six rows of `GET / — bot — 200` *is* the demo GIF, and it undersells the tool badly. It also means anyone who installs it sees an empty screen on first run.

So the demo dataset is not optional polish — it's what makes both the GIF and the first-run experience work. Build it early, from realistic shapes, and be transparent in the README that it's seed data.

Getting listed is a forcing function with a clean finish line. It is a weak quality signal on its own — the list accepts most working things. Build for the real bar; the listing follows.

---

## Resolved decisions

Output of a ten-round `/grill-me` session. Settled unless new evidence appears.

| # | Decision | Rationale |
|---|---|---|
| 1 | **"Local" = local database on the user's machine** | Events sync down; queries run against a local SQLite file. Not "UI is local, data lives in someone's cloud." |
| 2 | **Agent-readability is the hook** | A local file can be queried directly by an agent in the terminal — no API, no auth, no rate limits. Human TUI and agent read the same file. This is what separates it from every other dashboard on the list. |
| 3 | **Audience = devs running agents in the terminal** | Till's framing. Sharpest idea in the session. |
| 4 | **TUI, not a VS Code extension** | Editor-agnostic, works over SSH. Not "because people live in VS Code" — that argument points at an extension. |
| 5 | **Scroll depth is cut** | Marketer feature. Needs long content pages and per-page volume. Neither exists. |
| 6 | **Journey maps are cut** | Need ~500+ sessions/week to be anything but noise. Reframed to funnels, then cut too — a funnel at n=8 fails the same test. |
| 7 | **No custom collector, no self-hosting, no paying** | Do not build *or run* ingest infrastructure, and don't rent it either. ~~Cloudflare~~ → ~~Umami Cloud~~ → **GoatCounter** (see Phase 0 and the source decisions below). |
| 8 | **Streams, not aggregates** | Aggregates need volume. Streams don't. "Last 50 visits: what they hit, where from, what they did" is honest at any n, trivially a local table, and directly agent-queryable. |
| 9 | **Till is not the target user** | No personal site has meaningful traffic. Dogfooding unavailable — acceptable now that this is a portfolio piece. |
| 10 | **"Local" = the read path, not the storage of record** | Events live in the vendor's cloud; the local SQLite file is a synced copy. TUI and agent read only the local file. |
| 11 | **Name: `quinto`** | From *quinto sabor*, the fifth taste — umami. Named after the *category* of taste, not the vendor, which is why it survived the source swap unscathed. Short, typeable, no vowel-dropping. |
| 12 | **Session-grouped stream view** | One row per visit, expandable to the path through the site. The honest small-n form of the customer journey map from the original brief. |

---

## Assumptions (flag if wrong)

- Biggest personal site shows **68 unique visitors / 24h** in Cloudflare, but **245 total requests** — 3.6 req/visitor, flat overnight, peaking at 4 AM. **Likely mostly bots.** Real human sessions probably single digits/day. ← *unverified*
- Low volume is now a **feature constraint, not a blocker** — it forces the stream design, which is the more interesting build anyway.
- ~~Cloudflare's GraphQL API exposes enough per-request detail~~ — **disproven 2026-07-24.** Enterprise only.
- ~~Umami Cloud's free tier is usable~~ — **disproven 2026-07-25.** API access is €20/month. Its data model was fine; the price wasn't.
- GoatCounter's export carries session IDs and a bot flag — **verified 2026-07-25** against their CSV export docs.
- GoatCounter stays free for this usage. ← *"currently offered for free for reasonable public usage" is their wording — not a contract. Accepted risk.*

---

## Approach

### Phase 0 — RESOLVED 2026-07-24 ❌ Cloudflare cannot be the source

**Question:** can Cloudflare give per-request rows (Decision 8's stream view) on a non-Enterprise plan?

**Answer: no.** Every route to raw request-level data is Enterprise-gated:

| Route | Free | Pro | Business | Enterprise |
|---|---|---|---|---|
| Logpush | ❌ | ❌ | ❌ | ✅ |
| Logpull (legacy) | ❌ | ❌ | ❌ | ✅ |
| Instant Logs | ❌ | ❌ | ❌ | ✅ |

*(Source: Cloudflare Logs docs, availability table, retrieved 2026-07-24.)*

The GraphQL Analytics API is no way around it. Its zone HTTP and RUM nodes are all `…Groups` — aggregated into dimension buckets by design. Raw (non-`Groups`) nodes exist only for a few products, not zone HTTP traffic. **There is no per-visit row anywhere on a free plan.**

Cloudflare Web Analytics *is* free on all plans, but it's the same story: aggregated RUM groups, no session-level stream.

**Consequence:** Decision 7 stands (still not writing a collector), Decision 8 stands (streams are still the right shape), but the **source changes**.

### Umami Cloud — ❌ RULED OUT 2026-07-25

**The free plan does not include API access.** An API key requires the paid tier at €20/month — €240/year, for a portfolio project, to read your own data. The API *is* the entire point of the integration, so the free plan is worthless here.

Self-hosting Umami would restore the API for nothing, but that call was already made and stands: **no self-hosting.** It was made when hosted was free, which weakens it — except the option below is free *and* hosted, so the trade never has to be made.

### The pattern behind three failed sources

Cloudflare gates raw request logs behind Enterprise. Umami gates the API behind €20/month. **This is not bad luck — raw per-event export is precisely the feature analytics vendors charge for.** Aggregates are cheap to serve and cheap to give away; individual rows are the product. Any future source has to be evaluated on this axis first, before anything else.

The exception is software that isn't monetised through hosting at all.

### Source decision — GoatCounter. Verified 2026-07-25.

Open source, donation-supported, **free hosted service** — *"currently offered for free for reasonable public usage. Running your personal website or small-to-medium business on it is fine."* API keys are created in the account with no paid gate.

**`POST /api/v0/export` returns a CSV of individual pageviews**, with exactly the fields this project needs:

| Field | Why it matters here |
|---|---|
| `Session` | Session ID — **the session-grouped stream view survives intact** |
| `FirstVisit` | Marks the session's first hit; gives you session boundaries for free |
| `Bot` | Explicit bot classification — **you can finally exclude the crawler noise** that made the Cloudflare numbers meaningless |
| `Path`, `Title`, `Event` | Path doubles as event name |
| `Browser`, `System`, `Screen size` | Client detail |
| `Referrer`, `Referrer scheme` | Scheme distinguishes link / generated / campaign / other |
| `Location` | ISO 3166-2 country code |
| `Date` | RFC 3339 |

**It's better than Umami would have been, on three counts:**

1. **Bulk export kills the N+1.** One request produces the whole dataset, gzipped. No per-session fan-out.
2. **`last_hit_id` is a cursor.** `POST /api/v0/export` accepts `start_from_hit_id`, so incremental sync is a native feature — you don't invent a watermark, you store their cursor.
3. **The `Bot` column** turns the project's biggest data-quality problem into a `WHERE` clause.

Also: ~3.5 KB script, no cookies, no GDPR notice required, and a JS-free pixel fallback.

⚠️ **Setup gotcha that will silently break everything:** the export requires **"Individual pageviews" enabled in site settings — it is off by default.** Without it there is no per-hit data and the whole design fails. Turn it on before anything else.

⚠️ **The export format is versioned** — the version number is the first field of the CSV header, and the docs strongly recommend erroring out if it changes rather than mis-parsing. Note also that GoatCounter's own docs say the **JSON export** is the better option for most people; check it before committing to CSV parsing.

⚠️ **Rate limit: 4 requests/second.** Irrelevant at this volume, relevant if anyone points `quinto` at a large site.

⚠️ **Durability risk, stated plainly:** GoatCounter is one maintainer, donation-supported, and "currently free" is not a contract. That's a genuine risk — mitigated by the fact that the sync boundary makes swapping sources cheap. This is now the third source, and the architecture has absorbed all three without touching the TUI. That's evidence the boundary is in the right place.

### Sources evaluated — why not Plausible, Fathom, PostHog?

The requirement is narrow: **free, hosted, and raw per-visit rows with session identity.** Most analytics products fail the third one regardless of price.

| Source | Free? | Hosted? | Raw per-visit rows? | Verdict |
|---|---|---|---|---|
| **GoatCounter** | ✅ donation-supported | ✅ | ✅ export API, `Session` + `Bot` columns | **Chosen** |
| **PostHog Cloud** | ✅ generous free tier | ✅ | ✅ raw events via query API | Viable. Heavy bundle, consent surface, product-analytics complexity for a visit list. |
| **Plausible** | ❌ no free tier | ✅ | ❌ **aggregated only** | Ruled out — see below |
| **Fathom / Simple Analytics** | ❌ | ✅ | ❌ aggregated | Ruled out |
| **Umami Cloud** | ❌ API is €20/mo | ✅ | ✅ | Ruled out on price |
| **Cloudflare** | ✅ | ✅ | ❌ Enterprise only | Ruled out |

**Plausible deserves a specific note, because it's the obvious suggestion.** It is a better *product* than GoatCounter — nicer dashboard, more polish, a real company behind it. It is unusable *here*, for two independent reasons:

1. **The Stats API is a Business-plan feature** — not merely paid, but their upper tier.
2. **Even on that plan it's aggregated.** `/api/v2/query` takes `metrics` and `dimensions` and returns rollups. There is no per-visit row, so the session-grouped stream view — the whole product — cannot be built on it at any price.

Price is not why Plausible loses. **Shape is.** GoatCounter isn't the better analytics product; it's the one whose data-access model fits.

### Why not build the collector ourselves?

Reconsidered 2026-07-25 after three vendor failures — a legitimate prompt to revisit, not a settled question.

**The case for building it** is stronger than it looks: no fourth vendor to fail, "local" becomes literally true, total control of the schema (and the schema is the product), and at ten visits a day the collector is genuinely small. On Cloudflare Workers + D1 it's roughly a hundred lines with no server to maintain — *serverless is not self-hosting in the ops sense*, so the earlier "no self-hosting" call doesn't actually forbid it.

**The case against is distribution, and it decides it.** The definition of done is a stranger installing this in under a minute. With a vendor, anyone who already runs GoatCounter points `quinto` at it and has data immediately. With a bespoke collector, every user must first deploy *your backend* — which kills the install criterion, and with it the awesome-tuis pitch.

Secondary but real: bot classification is hard and GoatCounter gives it free — and bots are this project's single worst data-quality problem. Cookieless session identity (salted, daily-rotating hashes) is fiddly. GDPR posture becomes yours. And the two-weekend timebox does not survive any of it.

**Decision: GoatCounter for v1, and the question stays open rather than closed.** The sync boundary is the one thing that has survived three source swaps untouched, so `quinto collect` can be added later as another source behind the same interface, at near-zero cost to defer.

The honest reason not to build it first: most portfolio projects never reach v2. Spend the two weekends on the thing being judged.

**Sliced into tickets 2026-07-25** — see `.scratch/quinto-collect/issues/`. Six tickets: beacon-to-storage tracer, cookieless sessions, bot classification, abuse protection, export API with cursor, deploy story.

**The slicing is itself the argument.** The collector is six tickets against v1's six — *a second project the same size as the first*, and with more unknowns: bot classification accuracy, abuse handling on a public endpoint, and free-tier limits that have already killed three sources in this project. Two of the six (abuse protection, deploy story) produce no visible feature at all.

They're written as a separate effort deliberately. Two ways to use them:

- **As v2** — additive, behind the existing sync boundary. Ticket 05's success condition is that `quinto sync` swaps source with no change to the TUI, schema or query interface. That doubles as a test of whether the boundary is where this plan claims it is.
- **Instead of GoatCounter in v1** — then v1's ticket 01 is replaced by all six of these, the timebox roughly triples, and the under-a-minute install in ticket 06's definition of done breaks. Choose this only with that trade stated out loud.

### What this does to Decision 1 ("local")

The record of truth lives in GoatCounter; the local SQLite file is a **synced copy**. Worth being precise: **"local" describes the read path, not the storage of record.** The TUI and the agent both read the local file, never the API — which is what Decisions 1 and 2 actually depend on. Both survive intact.

## Architecture

```mermaid
flowchart TD
    V["Visitor's browser<br/>(goatcounter, ~3.5KB)"] -->|pageviews| G["GoatCounter<br/>storage of record"]
    G -->|"export API + cursor"| S["sync command<br/>YOU BUILD THIS"]
    S -->|writes| D[("quinto.db<br/>local file")]
    D --> T["TUI<br/>YOU BUILD THIS"]
    D --> A["Agent in the terminal"]
    F["demo fixture"] -.->|seeds| DD[("quinto-demo.db")]
    DD --> T
```

**The TUI never talks to GoatCounter.** Only `sync` does. That single decision buys instant rendering, offline use, and — because the file is just a file — direct agent access with no API, auth, or rate limit in the way. It's also what makes Decision 2 real rather than a slogan, and it's why swapping the source three times has cost nothing downstream.

### What you actually build

Three things. Everything else is configuration.

| Piece | What it does |
|---|---|
| `sync` | GoatCounter export → SQLite. Incremental via their `last_hit_id` cursor, idempotent on hit ID. |
| TUI | Reads SQLite with SQL. Stream view first, overview second. |
| `query` subcommand | Runs raw SQL against the local file and prints it. **This is the agent interface** — an agent needs no database install and no schema hunting. |

### Local schema (draft)

The schema *is* the product — it's what the agent reads. Optimise it for legibility to something that has never seen the codebase.

GoatCounter's export is flat — one row per hit, with `session` grouping them. That's simpler than a two-table model and better for the agent interface, so keep it flat and derive sessions as a view:

```sql
hits(hit_id PK, session, path, title, is_event,
     browser, system, bot, referrer, referrer_scheme,
     screen_size, country, first_visit, created_at)

sessions  -- a VIEW over hits, grouped by session:
          -- first_seen, last_seen, page_count, entry_path,
          -- referrer, country, browser
```

Maps 1:1 onto GoatCounter's export columns — no translation layer, no invented vocabulary. `bot` stays in the table rather than being filtered at sync time, so the exclusion is visible and reversible rather than a silent data loss.

### Setup sequence

1. Sign up at GoatCounter → you get `yourcode.goatcounter.com`
2. **Enable "Individual pageviews" in site settings.** ⚠️ Off by default. Without it there is no per-hit export and nothing below works.
3. Snippet into the site's `<head>` — data accumulates immediately:
   ```html
   <script data-goatcounter="https://yourcode.goatcounter.com/count"
           async src="//gc.zgo.at/count.js"></script>
   ```
4. Generate an **API key** (top menu → your username → API)
5. `quinto sync` → populates the local SQLite file
6. `quinto` → TUI reads the file
7. Agent: `quinto query "select path, count(*) from hits where bot = 0 group by 1"`

Steps 1–4 are an afternoon and require no code. The demo fixture writes the **same** schema to a separate file, so the TUI is identical whether it's showing real data or seed data — which is why it can be built before any real traffic arrives.

### Phase 1 — Build (the whole project)

**Timebox: 2 weekends to something demoable.** If it runs longer, the scope was wrong.

The work queue lives in **`.scratch/quinto-v1/issues/`** — six tracer-bullet tickets, each with its blocking edges. This section is the shape; the tickets are the detail.

| # | Ticket | Blocked by |
|---|---|---|
| 01 | Sync one real visit from GoatCounter to the terminal | — |
| 02 | Demo dataset | 01 |
| 03 | Stream view TUI | 01, 02 |
| 04 | `quinto query` — the agent interface | 01 |
| 05 | Overview / site health screen | 03 |
| 06 | Ship | 03, 04, 05 |

Two ordering choices that aren't arbitrary: **02 precedes 03** so the TUI is designed against realistic density rather than six real rows, and **04 depends only on 01** so the agent interface can be built in parallel with the TUI.

Ticket 01 additionally carries the repo skeleton, and is gated on a human prerequisite — a GoatCounter account with individual pageviews enabled and an API key — not on another ticket.

**Testing policy:** test the sync logic, not the rendering. Idempotency, cursor handling and export-format version checking are claims you can't verify by eye. TUI output gets looked at, not asserted on. Two weekends is not a budget for more, and this is where tests actually pay.

**Demo vs real data:** `quinto demo` writes to a *separate database file* selected by a flag — not a source column in the shared schema. The schema is the agent interface and stays free of internal bookkeeping, and demo data can't clobber real data by construction.

### Cut from the plan

Netnography, interviews, kill criteria, segment validation. All correct for a product; all waste for a portfolio piece. Deleted on purpose — if this ever turns back into a product, restore them from git history.

---

## Rejected

| Rejected | Why |
|---|---|
| Own tracking script + ingest + event store | The TUI becomes step 5 of 5 and ~20% of the work. Not the point. |
| VS Code extension | Abandons Neovim/Zed/SSH users. |
| "For people in VS Code" positioning | Retrofitted rationale. |
| mctimey.app as dogfood target | No volume. |
| Scroll depth, journey maps, funnels | Wrong audience for a terminal; unbuildable at this volume. |
| Validating the segment before building | Right for a product, waste for a portfolio project. |

---

## Offen — bei Till

- [x] ~~Phase 0 check — can Cloudflare give per-request rows?~~ **No. Enterprise only.** Source switched to Umami.
- [x] ~~Umami self-hosted vs PostHog~~ **Umami Cloud. No self-hosting.** (2026-07-25)
- [x] ~~Sign up for Umami Cloud~~ **Abandoned 2026-07-25 — API costs €20/month.**
- [ ] **Sign up at GoatCounter**, then **enable "Individual pageviews" in site settings** — off by default, and nothing works without it.
- [ ] **Paste the GoatCounter snippet into the site's `<head>`.** Nothing is blocked by it, but every day without it is a day of history you can't get back.
- [ ] **Generate a GoatCounter API key** — blocks ticket 01.
- [x] ~~Smallest useful stream view~~ **Session-grouped, expandable** (2026-07-25). One row per visit, expands to the path through the site. The honest small-n form of the customer journey map from the original brief.
- [x] ~~Break the build into tickets~~ **Six tracer bullets** in `.scratch/quinto-v1/issues/` (2026-07-25).
- [x] ~~Name~~ **`quinto`** (2026-07-25). Check availability before publishing: GitHub org/repo, Homebrew formula, and the `quinto` name on any registry you'd publish to.
- [ ] Framework: **Go (Bubble Tea)** unless you have Rust. Not really open — the single-binary promise rules out Ink, and Bubble Tea has the richest component set for dashboards. Confirm and move on.
- [ ] Optional: verify the 68 in Cloudflare's bot breakdown. Doesn't block anything, but tells you what real data will look like next to your seed data.

---

## Notes

- Output of a `/grill-me` session, 2026-07-24. The plan changed substantially: audience changed, medium got an honest justification, the feature set inverted, "local" became an architecture, and the project reclassified from product to portfolio.
- **Watch:** Claude authored more of this concept than Till did. Decision 2 (agent-readability) came from Claude, not from user evidence. It's the strongest differentiator on paper and also the least tested — treat as hypothesis.
