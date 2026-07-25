# 05 — Overview / site health screen

**What to build:** A second screen showing the standard numbers — visitors over time, top pages, top referrers, and a traffic trend for the selected period — reachable from the stream view by keyboard.

This is the "how is the site doing" glance, as opposed to the stream's "what did this person do". It's the part of the original brief that survives at any traffic level, because unlike journeys and funnels these numbers stay meaningful when n is small.

Keep it honest about low volume. A chart drawn from four data points should look like four data points, not a smooth trend line implying precision that isn't there.

**Blocked by:** 03 — needs the TUI shell and its navigation model.

**Status:** resolved (2026-07-25)

- [x] A second screen is reachable from the stream view and back again, by keyboard
- [x] Shows visitors and pageviews over a selectable time range
- [x] Shows top pages and top referrers
- [x] Shows a traffic trend over the selected range
- [x] Sparse data renders honestly rather than being smoothed into a false trend
- [x] Time range selection is shared between screens, so switching views keeps the period
- [x] Reads only the local database — no network

## Answer

Closed 2026-07-25. Reachable with `tab` from the stream view; `r` cycles
7 days / 28 days / all time, and the range is shared so switching screens never
silently changes the period.

```
quinto · DEMO DATA · synced 21m ago · 309 visits · 707 pageviews · 111 bot visits hidden

  73 visitors   130 pageviews   40 events   42/73 single-page   24 bots hidden

  pageviews per day · 7 days
  ▆▆▄▆▃█▃▂
  peak 33/day · 8 of 8 days had traffic

  top pages                     referrers                     countries
  /                          53 direct                     39 DE                    23
  /work                      19 Google                     11 US                    11
  /writing                   17 Hacker News                10 GB                    10
```

**Honesty rules the chart follows**, because this screen is where a low-traffic
tool is most tempted to flatter:

- A day with no traffic draws **nothing**, not a baseline block. Drawing
  something for an empty day implies activity that did not happen.
- Below five days with data the chart says *"too few days to call it a trend"*
  rather than presenting noise as a line.
- Bounces are a **count over their denominator** (`42/73`), never a percentage.
  A rate implies a precision the sample cannot support.
- Hidden bots are still counted in the header.

One bug worth recording: styling the numbers inside the column builder embedded
ANSI escapes that the padding function then counted as visible characters,
quietly misaligning every column. Column content is now assembled as plain text
and styled only after padding.
