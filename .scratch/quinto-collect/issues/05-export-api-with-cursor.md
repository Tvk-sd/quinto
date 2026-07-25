# 05 — Export API with cursor

**What to build:** An authenticated read endpoint that hands `quinto sync` the hits it doesn't have yet, so the collector becomes a drop-in source alongside GoatCounter.

The success condition is precise: **`quinto sync` pulls from your own collector and nothing in the TUI, the schema or the query interface changes.** If any of them need touching, the sync boundary wasn't where it should be — which would be a genuinely useful finding, since that boundary is the architectural claim this project rests on.

Note what this ticket really is: you are reimplementing the export API that GoatCounter already gives you for free — token auth, incremental cursor, bulk transfer, format versioning. That's not an argument against building it, but it should be a conscious one.

**Blocked by:** 01, 02, 03 — the export is only worth building once the rows have sessions and bot flags in them.

**Status:** ready-for-agent

- [ ] An authenticated endpoint returns stored hits
- [ ] Requests without a valid token are rejected
- [ ] A cursor lets a caller fetch only what it hasn't seen, without re-reading history
- [ ] Large result sets transfer in bounded chunks rather than one unbounded response
- [ ] The response carries a format version so consumers can fail loudly on a breaking change
- [ ] `quinto sync` pulls from this collector with no change to the TUI, schema or query interface
- [ ] Switching a user between GoatCounter and this collector is a configuration change, not a code change
