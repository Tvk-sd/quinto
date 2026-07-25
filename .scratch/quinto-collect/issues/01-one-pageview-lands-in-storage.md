# 01 — One pageview lands in storage

**What to build:** You load a page in a browser and a row appears in a database you own. A minimal beacon script fires on pageview, a Cloudflare Worker receives it, and the hit is written to D1. Nothing else — no sessions, no geography, no bot detection.

This is the tracer bullet for the whole collector: it proves the beacon fires, the endpoint is reachable, and storage works, before any of the hard parts are attempted.

**Settle the feasibility question first, before writing the Worker.** This project has already lost three data sources to gating discovered too late. Confirm against current Cloudflare docs that the free tiers actually carry this: Workers requests per day, D1 rows written per day, D1 database size, and whether a custom domain or route is needed. If any limit makes this unworkable at a realistic traffic level, that is a finding worth stopping on — write it down and raise it before continuing.

The beacon must stay small. Its size is a feature of the product, and anything over a couple of kilobytes undermines the reason for choosing a lightweight source in the first place.

**Blocked by:** None — can start immediately.

**Status:** ready-for-agent

- [ ] Cloudflare Workers and D1 free-tier limits are verified against current docs and written down
- [ ] A beacon script fires once per pageview and sends path, title, referrer, screen size and timestamp
- [ ] A Worker receives the beacon and writes one row per hit to D1
- [ ] The row is visible by querying D1 directly
- [ ] The beacon handles client-side navigation in single-page apps, not just full page loads
- [ ] The beacon is under 2 KB and does not block page rendering
- [ ] No cookies are set and no IP address is stored
