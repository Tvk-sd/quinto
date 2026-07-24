# TUI Analytics — Plan

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
- [ ] One hook that isn't "another pretty dashboard." Current candidate: the local data file is agent-queryable.
- [ ] Runs on someone else's machine in under a minute — single binary or one install command.
- [ ] A name worth typing.

Getting listed is a forcing function with a clean finish line. It is a weak quality signal on its own — the list accepts most working things. Build for the real bar; the listing follows.

---

## Resolved decisions

Output of a ten-round `/grill-me` session. Settled unless new evidence appears.

| # | Decision | Rationale |
|---|---|---|
| 1 | **"Local" = local database on the user's machine** | Events sync down; queries run against a local DuckDB/SQLite file. Not "UI is local, data lives in someone's cloud." |
| 2 | **Agent-readability is the hook** | A local file can be queried directly by an agent in the terminal — no API, no auth, no rate limits. Human TUI and agent read the same file. This is what separates it from every other dashboard on the list. |
| 3 | **Audience = devs running agents in the terminal** | Till's framing. Sharpest idea in the session. |
| 4 | **TUI, not a VS Code extension** | Editor-agnostic, works over SSH. Not "because people live in VS Code" — that argument points at an extension. |
| 5 | **Scroll depth is cut** | Marketer feature. Needs long content pages and per-page volume. Neither exists. |
| 6 | **Journey maps are cut** | Need ~500+ sessions/week to be anything but noise. Reframed to funnels, then cut too — a funnel at n=8 fails the same test. |
| 7 | **No custom collector** | Do not build ingest infrastructure. Cloudflare's GraphQL API is already available and needs no snippet. |
| 8 | **Streams, not aggregates** | Aggregates need volume. Streams don't. "Last 50 visits: what they hit, where from, what they did" is honest at any n, trivially a local table, and directly agent-queryable. |
| 9 | **Till is not the target user** | No personal site has meaningful traffic. Dogfooding unavailable — acceptable now that this is a portfolio piece. |

---

## Assumptions (flag if wrong)

- Biggest personal site shows **68 unique visitors / 24h** in Cloudflare, but **245 total requests** — 3.6 req/visitor, flat overnight, peaking at 4 AM. **Likely mostly bots.** Real human sessions probably single digits/day. ← *unverified*
- Low volume is now a **feature constraint, not a blocker** — it forces the stream design, which is the more interesting build anyway.
- Cloudflare's GraphQL API exposes enough per-request detail to build a useful stream view. ← *unverified, check before committing to it*

---

## Approach

### Phase 1 — Build (the whole project)

**Timebox: 2 weekends to something demoable.** If it runs longer, the scope was wrong.

1. Verify Cloudflare's GraphQL API returns per-request rows with enough fields (path, referrer, country, UA, status, timing). If not, fall back to a self-hosted collector or Umami.
2. Local store: DuckDB or SQLite. **The schema is the product** — it's what the agent reads. Design it to be legible to something that has never seen the codebase.
3. Sync command: pull → local file. Incremental.
4. TUI: stream view first. Overview/health second. Nothing else until both feel good.
5. Demo GIF, README, install path.
6. PR to awesome-tuis.

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

- [ ] **Smallest useful stream view** — what would you actually want on screen? Defines the 2-weekend timebox.
- [ ] **Name.** Blocks the repo.
- [ ] Language/framework: Go (Bubble Tea) vs Rust (Ratatui) vs TS (Ink). Affects the single-binary install promise — Ink does not deliver it.
- [ ] Optional: verify the 68 in Cloudflare's bot breakdown. Doesn't block anything now, but tells you what your demo data will actually look like.

---

## Notes

- Output of a `/grill-me` session, 2026-07-24. The plan changed substantially: audience changed, medium got an honest justification, the feature set inverted, "local" became an architecture, and the project reclassified from product to portfolio.
- **Watch:** Claude authored more of this concept than Till did. Decision 2 (agent-readability) came from Claude, not from user evidence. It's the strongest differentiator on paper and also the least tested — treat as hypothesis.
