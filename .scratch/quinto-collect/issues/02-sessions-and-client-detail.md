# 02 — Sessions, geography and client detail

**What to build:** Two pageviews from the same visitor are recognisably the same visit, and each hit carries country, browser and operating system. Without this the session-grouped stream view — the whole point of the product — has nothing to group by.

**Session identity without cookies is the hard part of this ticket.** The standard approach is a hash of IP address, user agent and a rotating salt, where the salt changes daily so visits cannot be linked across days. That gives you session identity with nothing personal stored and no consent banner required. Its known quirk: **sessions break at the salt rotation boundary**, so a visit spanning midnight appears as two. Document that rather than hiding it.

Geography and client detail are close to free on this platform — Cloudflare exposes country on the incoming request, and browser and system parse from the user agent. Take the cheap wins here; they are what makes a stream row readable.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] Two pageviews from one visitor in one sitting share a session identifier
- [ ] Two different visitors never share a session identifier
- [ ] Session identity uses a rotating salt so visits cannot be linked across days
- [ ] No IP address or raw user agent is persisted
- [ ] Each hit records country, browser and operating system
- [ ] The first hit of a session is marked as such
- [ ] Salt rotation behaviour and its midnight-boundary effect are documented
- [ ] Session derivation is covered by tests
