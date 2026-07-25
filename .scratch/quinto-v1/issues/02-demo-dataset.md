# 02 — Demo dataset

**What to build:** `quinto demo` fills the local database with realistic fake traffic, so the tool has something worth looking at without a live GoatCounter account or real visitors.

This exists for three reasons, and all three matter:

1. **The demo GIF.** Real traffic is single-digit visits a day, mostly bots. A stream view showing six rows undersells the tool badly, and the GIF is what gets it noticed.
2. **First run.** Anyone who installs it should see a populated tool, not an empty screen.
3. **Design density.** The stream view has to be built against realistic volume. Designing against six rows produces something that looks broken at five hundred.

The data has to be plausible, not random: sessions with coherent paths through a site, referrers that make sense together, a believable spread of geography and browsers, and a diurnal traffic curve rather than a flat line. Seeded so it's reproducible.

It writes the same schema as ticket 01 but into a **separate database file**, selected by a flag. That keeps the two cleanly apart without adding a source column to the schema — and the schema is the agent interface, so it stays free of bookkeeping that only matters internally. It also makes "demo can't clobber real data" true by construction rather than by a guard.

**Blocked by:** 01 — needs the schema to exist.

**Status:** resolved (2026-07-25)

- [x] `quinto demo` populates the local database with several hundred sessions across a multi-week range
- [x] Sessions contain coherent multi-page paths, not disconnected random hits
- [x] Referrers, geography, browsers and session durations are plausibly distributed
- [x] Traffic follows a day/night curve rather than being uniform
- [x] Generation is seeded and reproducible
- [x] Demo data is written to a separate database file, never to the real one
- [x] The TUI and `query` can be pointed at the demo file by a flag
- [x] While viewing demo data the interface says so, so nobody mistakes it for their own traffic

## Answer

Closed 2026-07-25. `quinto demo` generates 420 sessions / ~820 hits over 28
days into a separate `quinto-demo.db`; `quinto --demo` reads it and labels
every screen `DEMO DATA — generated sample traffic, not a real site`.

Deliberately unflattering, because a demo that flatters the tool is a sales
pitch rather than a preview:

```
bot share            26%      (crawlers really are a third of small-site traffic)
single-page visits   60%      (most people bounce)
sessions with 4+     22       (enough to demonstrate the expandable journey)
overnight traffic    present but ~15x below the midday peak
```

**Two modelling errors only surfaced once realistic data existed**, which is
the argument for building this before the TUI rather than after:

1. `page_count` counted events as pages. With events on most pages that error
   is systematic — it turned bounces into journeys and made two-page visits
   look nearly as common as one-page ones. Pages and events are now counted
   separately.
2. An event is a second *observation*, so a single-page visit that fires one
   has a measurable duration after all. 68 of 185 single-page demo visits are
   bounded this way; the rest correctly stay NULL.

Also fixed a display bug the multi-day data exposed: the recent list showed
only `HH:MM`, so visits from different days appeared out of order.
