# 04 — Abuse protection

**What to build:** The ingest endpoint survives contact with the open internet.

**This ticket is not optional and it is the one people skip.** A collector endpoint is public by definition — anyone who views source on the site can see where the beacon posts. Without protection, a single script can flood your database with fabricated pageviews, exhaust the free tier in an afternoon, and poison every number the tool displays. A vendor absorbs this problem for you; building your own means owning it.

At minimum the endpoint should reject traffic that didn't come from a site you configured, cap how fast any one source can write, and refuse payloads that are malformed or oversized. When limits are hit it should fail cheaply rather than doing expensive work before rejecting.

None of this is exotic, but all of it is work that produces no visible feature — which is exactly why it gets deferred and then never done.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] Requests from origins not configured for the site are rejected
- [ ] Any single source is rate limited, and exceeding the limit does not consume storage
- [ ] Malformed or oversized payloads are rejected before any database write
- [ ] Field lengths are bounded, so a hostile client cannot store arbitrary data
- [ ] Rejections are cheap — no expensive processing happens before the reject decision
- [ ] The endpoint's behaviour under a flood is tested, not assumed
- [ ] Free-tier consumption under abuse is estimated and documented
