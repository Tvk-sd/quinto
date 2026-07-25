# 05 — Overview / site health screen

**What to build:** A second screen showing the standard numbers — visitors over time, top pages, top referrers, and a traffic trend for the selected period — reachable from the stream view by keyboard.

This is the "how is the site doing" glance, as opposed to the stream's "what did this person do". It's the part of the original brief that survives at any traffic level, because unlike journeys and funnels these numbers stay meaningful when n is small.

Keep it honest about low volume. A chart drawn from four data points should look like four data points, not a smooth trend line implying precision that isn't there.

**Blocked by:** 03 — needs the TUI shell and its navigation model.

**Status:** ready-for-agent

- [ ] A second screen is reachable from the stream view and back again, by keyboard
- [ ] Shows visitors and pageviews over a selectable time range
- [ ] Shows top pages and top referrers
- [ ] Shows a traffic trend over the selected range
- [ ] Sparse data renders honestly rather than being smoothed into a false trend
- [ ] Time range selection is shared between screens, so switching views keeps the period
- [ ] Reads only the local database — no network
