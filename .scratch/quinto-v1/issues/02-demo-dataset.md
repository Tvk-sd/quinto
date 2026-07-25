# 02 — Demo dataset

**What to build:** `quinto demo` fills the local database with realistic fake traffic, so the tool has something worth looking at without a live GoatCounter account or real visitors.

This exists for three reasons, and all three matter:

1. **The demo GIF.** Real traffic is single-digit visits a day, mostly bots. A stream view showing six rows undersells the tool badly, and the GIF is what gets it noticed.
2. **First run.** Anyone who installs it should see a populated tool, not an empty screen.
3. **Design density.** The stream view has to be built against realistic volume. Designing against six rows produces something that looks broken at five hundred.

The data has to be plausible, not random: sessions with coherent paths through a site, referrers that make sense together, a believable spread of geography and browsers, and a diurnal traffic curve rather than a flat line. Seeded so it's reproducible.

It writes the same schema as ticket 01 but into a **separate database file**, selected by a flag. That keeps the two cleanly apart without adding a source column to the schema — and the schema is the agent interface, so it stays free of bookkeeping that only matters internally. It also makes "demo can't clobber real data" true by construction rather than by a guard.

**Blocked by:** 01 — needs the schema to exist.

**Status:** ready-for-agent

- [ ] `quinto demo` populates the local database with several hundred sessions across a multi-week range
- [ ] Sessions contain coherent multi-page paths, not disconnected random hits
- [ ] Referrers, geography, browsers and session durations are plausibly distributed
- [ ] Traffic follows a day/night curve rather than being uniform
- [ ] Generation is seeded and reproducible
- [ ] Demo data is written to a separate database file, never to the real one
- [ ] The TUI and `query` can be pointed at the demo file by a flag
- [ ] While viewing demo data the interface says so, so nobody mistakes it for their own traffic
