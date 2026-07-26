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

**Status:** ready-for-agent

## Done when

Behaviour:

- [ ] `/` opens a filter; typing narrows the list live; `enter` commits and returns to browsing
- [ ] Matches on landing page, referrer, country and browser, **and on every page inside the
      visit** — `/contact` must find visits where `/contact` was the second or third page
- [ ] A row whose match is not visible on the row states why it matched
- [ ] The highlight stays on the visit you were reading; if that visit stops matching, it goes
      to the top of the results — never to whatever now occupies the old row number
- [ ] Expanded visits stay expanded across a filter change
- [ ] `esc` clears a committed filter; with no filter it quits. `q` and `ctrl+c` always quit
- [ ] The bots toggle composes with the filter instead of resetting it, and keeps your place
      (this reverses `tui.go:176`, which resets the cursor to 0 today)
- [ ] While typing, single-letter bindings are text, not commands — the footer says so
- [ ] Header or footer shows the active filter and how many visits match

Constraints:

- [ ] **No new dependency.** Hand-roll the single-line input; do not add `bubbles`. Substring
      matching is fine for v1 — `sahilm/fuzzy` is the settled answer if fuzzy is ever wanted,
      and swapping it changes nothing structural
- [ ] **`quinto query` parity.** The active filter is expressible as one SQL statement. The
      deep-scope form is an `EXISTS` subquery into `hits`, index-backed by `hits_session`;
      measured at 0.021s → 0.023s on the demo dataset
- [ ] Loading every visit's hits up front replaces the lazy load at `tui.go:215`
- [ ] `?` help lists the new bindings
- [ ] No cgo

Tests — each decided rule gets one, because they are design commitments and every one of
them was arguable:

- [ ] Cursor keeps the visit when it survives the filter
- [ ] Cursor goes to the top when the anchored visit is filtered away — the single-step case
      (sit on a bot visit, toggle bots) rather than a typed filter, which hides it by
      narrowing progressively
- [ ] A visit matches on an inner page that its row does not show
- [ ] Expanded state survives a filter change
- [ ] `esc` clears before it quits

## While you are in there

The empty state offers *"N bot visits are hidden — press b"* (`tui.go:436`) whenever the list
is empty. Once a filter exists that is misleading: pressing `b` helps only if a hidden bot
visit matches the current query. Honest today because the bots toggle is the only filter.
Make the hint conditional, or drop it when a filter is active.

## Not in scope

Command palette (`:`), drill-down from the overview, remappable keys, and a theme file. All
are separate items on the same research note and none of them blocks this.
