# 02 — Demo dataset

**What to build:** `quinto demo` fills the local database with realistic fake traffic, so the tool has something worth looking at without a live Umami account or real visitors.

This exists for three reasons, and all three matter:

1. **The demo GIF.** Real traffic is single-digit visits a day, mostly bots. A stream view showing six rows undersells the tool badly, and the GIF is what gets it noticed.
2. **First run.** Anyone who installs it should see a populated tool, not an empty screen.
3. **Design density.** The stream view has to be built against realistic volume. Designing against six rows produces something that looks broken at five hundred.

The data has to be plausible, not random: sessions with coherent paths through a site, referrers that make sense together, a believable spread of geography and browsers, and a diurnal traffic curve rather than a flat line. Seeded so it's reproducible.

It writes into the same schema as ticket 01, so every downstream screen is identical whether it's showing real or seeded data.

**Blocked by:** 01 — needs the schema to exist.

**Status:** ready-for-agent

- [ ] `quinto demo` populates the local database with several hundred sessions across a multi-week range
- [ ] Sessions contain coherent multi-page paths, not disconnected random hits
- [ ] Referrers, geography, browsers and session durations are plausibly distributed
- [ ] Traffic follows a day/night curve rather than being uniform
- [ ] Generation is seeded and reproducible
- [ ] Demo data is visibly distinguishable from real data, so nobody mistakes it for their own traffic
- [ ] Running it does not destroy previously synced real data without an explicit confirmation
