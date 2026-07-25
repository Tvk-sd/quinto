# 03 — Stream view TUI

**What to build:** Running `quinto` opens a full-screen terminal view of recent traffic, grouped by session and expandable to reveal the path each visitor took through the site.

This is the core screen and the reason the project exists. It reads the local database directly and never touches the network.

Shape decided during ticket breakdown — one row per visit, expandable into the journey:

```
  ▼ 14:22  DE · Chrome · google.com     3 pages · 2m14s
      14:22  /
      14:23  /pricing
      14:24  /signup

  ▶ 14:19  US · Safari · direct           1 page  · 8s

  ▶ 14:03  GB · Firefox · news.yc.com     5 pages · 6m02s
```

**Why session-grouped and not a flat list:** at low traffic the only real story in the data is what a single visitor did. Aggregates need volume that this tool's users mostly won't have. This is the honest, small-n form of a customer journey map — the original ask, rendered in the only way that isn't a lie at n=1.

Build it against the demo dataset so it's designed for realistic density, then verify against real synced data.

**Blocked by:** 01, 02.

**Status:** resolved (2026-07-25)

- [x] `quinto` opens a full-screen view listing recent sessions, newest first
- [x] Each session row shows time, geography, browser, referrer, page count and duration
- [x] A session expands and collapses to show the ordered path of pages visited
- [x] Keyboard navigation moves between sessions; expand/collapse and quit are discoverable without documentation
- [x] The view adapts to terminal width and height, including small windows
- [x] It renders correctly with hundreds of sessions and with a nearly empty database
- [x] An empty database shows a message explaining how to get data, not a blank screen
- [x] Bots are excluded by default and can be toggled back into view
- [x] Opening the view performs no network requests

## Answer

Closed 2026-07-25. Bubble Tea + Lipgloss, adaptive colours so it reads on light
and dark terminals. Reads the local file only — no network on any keystroke.

```
quinto · DEMO DATA · synced 18m ago · 309 visits · 707 pageviews · 111 bot visits hidden
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
24/309  ↑↓ move · enter expand · b bots · ? help · q quit
```

That expanded block is the customer journey map from the original brief,
rendered in the only form that is honest at this traffic level.

**Three defects the rendered output exposed that tests alone had not:**

1. Expanding a visit low in the list left its journey below the fold —
   scrolling followed the cursor line, not the whole block. Now regression
   tested.
2. The header and footer overflowed narrow terminals and wrapped, destroying
   the layout. Both now drop detail progressively instead.
3. A list spanning several days showed only `HH:MM`, so visits read as out of
   order. Same-day visits show a clock, this week shows a weekday, older shows
   a date.

**Distinctions the view refuses to collapse**, because they carry meaning:
an unmeasurable duration renders as `—` and never `0s`; an unknown referrer
renders as `?` and never `direct`; bots are hidden by default but their count
is always stated, so the exclusion is visible rather than silent.

`quinto list` remains as a plain-table, pipeable alternative for scripts and
non-interactive terminals.
