# 01 — `/` filter for the stream view

**What to build:** A live text filter on the stream view, opened with `/`, narrowing the
visit list as you type.

Today the only way to narrow the stream is `b`, which hides bots. `/` is the convention in
4 of the 6 dashboard TUIs surveyed, and it is the single biggest gap between quinto and a
dashboard people call good.

**How it behaves is already decided** — see `../2026-07-26-filter-decision.md`. A throwaway
prototype settled four questions that looked like implementation detail and were not. Do not
re-open them while building; the reasoning and the measurements are on branch
`proto/filter-state-model` (`bed3725`), runnable with `make proto-filter`.

| Decision | Answer |
|---|---|
| Cursor | Keep the visit you were looking at — and if it is gone, go to the top. |
| Scope | Match every page inside the visit, not only what the row displays. |
| Expanded | Leave opened visits open across a filter change. |
| Esc | Clear the filter; quit when there is nothing to clear. |

`streamfilter.Reduce` / `afterChange` on that branch are the validated shape. The reducer is
pure and has no I/O — it is meant to lift, not to be re-derived.

**Blocked by:** nothing. 03 and 05 are both closed.

**Status:** resolved (2026-07-26)

## Done when

Behaviour:

- [x] `/` opens a filter; typing narrows the list live; `enter` commits and returns to browsing
- [x] Matches on landing page, referrer, country and browser, **and on every page inside the
      visit** — `/contact` must find visits where `/contact` was the second or third page
- [x] A row whose match is not visible on the row states why it matched
- [x] The highlight stays on the visit you were reading; if that visit stops matching, it goes
      to the top of the results — never to whatever now occupies the old row number
- [x] Expanded visits stay expanded across a filter change
- [x] `esc` clears a committed filter; with no filter it quits. `q` and `ctrl+c` always quit
- [x] The bots toggle composes with the filter instead of resetting it, and keeps your place
      (this reverses `tui.go:176`, which resets the cursor to 0 today)
- [x] While typing, single-letter bindings are text, not commands — the footer says so
- [x] Header or footer shows the active filter and how many visits match

Constraints:

- [x] **No new dependency.** Hand-roll the single-line input; do not add `bubbles`. Substring
      matching is fine for v1 — `sahilm/fuzzy` is the settled answer if fuzzy is ever wanted,
      and swapping it changes nothing structural
- [x] **`quinto query` parity.** The active filter is expressible as one SQL statement. The
      deep-scope form is an `EXISTS` subquery into `hits`, index-backed by `hits_session`;
      measured at 0.021s → 0.023s on the demo dataset
- [ ] ~~Loading every visit's hits up front replaces the lazy load at `tui.go:215`~~
      **Not done, deliberately — see the Answer below.**
- [x] `?` help lists the new bindings
- [x] No cgo

Tests — each decided rule gets one, because they are design commitments and every one of
them was arguable:

- [x] Cursor keeps the visit when it survives the filter
- [x] Cursor goes to the top when the anchored visit is filtered away — the single-step case
      (sit on a bot visit, toggle bots) rather than a typed filter, which hides it by
      narrowing progressively
- [x] A visit matches on an inner page that its row does not show
- [x] Expanded state survives a filter change
- [x] `esc` clears before it quits

## While you are in there

The empty state offers *"N bot visits are hidden — press b"* (`tui.go:436`) whenever the list
is empty. Once a filter exists that is misleading: pressing `b` helps only if a hidden bot
visit matches the current query. Honest today because the bots toggle is the only filter.
Make the hint conditional, or drop it when a filter is active.

## Not in scope

Command palette (`:`), drill-down from the overview, remappable keys, and a theme file. All
are separate items on the same research note and none of them blocks this.

## Answer

Closed 2026-07-26. `/` filters the stream; `esc` clears it.

```
  ▶ 11:00  DE · Chrome · Hacker News  ← /contact          2 pages · 30s
/contact  1 matching  ↑↓ move · enter expand · esc clear filter · ? help · q quit
```

**Filtering happens in SQL, not in Go.** `store.SessionFilter.SQL` is one builder
producing one statement, and `RecentSessions` is its only caller — so the list on screen
is always the result of a query someone could have typed themselves. That is the
`quinto query` parity criterion satisfied structurally rather than by keeping two
implementations in agreement by hand.

**One criterion was not met as written.** The ticket said to load every visit's hits up
front, replacing the lazy load at `tui.go:215`. That requirement came from the prototype,
which filtered in Go and therefore needed every page in memory. Filtering in SQL does not:
the `EXISTS` subquery does the matching in the database, and a correlated subquery returns
the matched page for the row to display. Lazy hit loading stays, memory stays bounded by
what is expanded rather than by the size of the database, and the behaviour the criterion
existed to produce is unchanged. Flagged rather than silently dropped.

The cursor rule got the test it needed. `TestLostAnchorGoesToTheTopNotTheOldRowNumber`
uses a fixture where the top row and the old row number are deliberately *different*
visits — sit on the bot at row 3, hide bots, and the old row number holds `keep` while the
top holds `workA`. Verified by mutation: restoring the old `if m.cursor >= len(...)` clamp
makes it fail with `cursor = 2 (keep), want row 0 (workA)`. Without that fixture design the
test would have passed under both behaviours and proved nothing.

Two things worth recording:

- **Wildcards had to be escaped.** Typing `%` in an unescaped `LIKE` matches every visit,
  which reads as a broken filter rather than a working one. `escapeLike` neutralises
  `%`, `_` and `\`, with a test.
- **The empty state used to lie.** It offered *"N bot visits are hidden — press b"*
  whenever the list was empty. With a filter active that is only true if a hidden bot
  visit matches the query, so the hint is now counted and conditional — it says
  *"3 hidden bot visits do match"* or stays quiet.

No new dependency: the input is hand-rolled, matching is substring, `go.mod` is unchanged
and all four release targets still cross-compile with `CGO_ENABLED=0`.
