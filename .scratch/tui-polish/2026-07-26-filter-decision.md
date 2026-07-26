# DECIDED — how the `/` filter behaves

**Status:** behaviour settled 2026-07-26. **Not built yet.**
**Primary source:** branch `proto/filter-state-model` (commit `bed3725`) — a throwaway
prototype, deliberately off main. `git checkout proto/filter-state-model && make proto-filter`.

Item 4 of `2026-07-25-tui-design-research.md` was costed "medium" with three things open:
filter in SQL or in Go, which fields are matchable, and what filtering does to the cursor
and expansion state. A logic prototype answered all three by making them drivable.

---

## The answer

Held in code as `streamfilter.Decided()` on that branch.

| Decision | Answer |
|---|---|
| **Cursor** | Keep the visit you were looking at — **and if it is gone, go to the top.** |
| **Scope** | Match every page inside the visit, not only what the row displays. |
| **Expanded** | Leave opened visits open across a filter change. |
| **Esc** | Clear the filter; quit when there is nothing to clear. |

Only the scope decision was settled by evidence. The rest were judgement, and two of them
barely mattered — when a decision's answers never disagree however you drive them, that is
itself the finding.

## The two findings worth more than the verdict

**1. Row-only matching is broken, and the cost of fixing it is imaginary.**
`/contact` was visited 24 times in the demo dataset and *never* as a landing page. Matching
only what the row shows finds **zero**. The objection was that matching inner pages means
abandoning lazy hit loading (`tui.go:215`) — measured at **0.021s → 0.023s**, against the
`hits_session` index that already exists. It stays a single SQL statement, so `quinto query`
parity holds.

**2. "Keep the visit" was incomplete as stated.**
It says nothing about the case where that visit is filtered away, and the fallback was the
row *index* — silently reinstating the option that had just been rejected. Sit on a bot
visit, press `b`, and you land on an unrelated human visit chosen by row number alone.

This only surfaced by constructing a single-step anchor destruction. Typing a filter narrows
progressively, so the cursor clamps toward the top on its own and the bug hides. On the
branch:

```sh
go run ./cmd/quinto-proto-filter --frame 'bjjb'   # anchor destroyed → row 1
go run ./cmd/quinto-proto-filter --frame 'bjjjb'  # anchor survives  → same visit kept
```

## What is left

Building it. That is a ticket, not a fold — the prototype settled *how it behaves*, not the
code. It needs: a hand-rolled single-line text input (no `bubbles` dependency), filter state
on `Model`, the deep-scope query with its `EXISTS` subquery, the cursor rule above, `esc`
handling, and footer plus help updates. `streamfilter`'s reducer is the part that lifts.

Two constraints carried over from the research notes:

- **No new dependency was added** to reach this decision. The matcher is deliberately dumb
  substring; the genre answer (`sahilm/fuzzy`, used by k9s, lazygit and gh-dash) is already
  settled and does not touch the state model.
- **`quinto query` parity.** Each filter must stay expressible as one query, or "the same
  file is queryable by agents" quietly erodes.

## Also noted, not acted on

The empty state offers *"N bot visits are hidden — press b"* (`tui.go:436`) even when the
filter is what emptied the list, and pressing `b` would not help unless a hidden bot visit
matches the current query. Honest today, because the bots toggle is the only filter. A
second filter makes it conditional.
