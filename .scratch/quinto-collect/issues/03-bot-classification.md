# 03 — Bot classification

**What to build:** Each hit is classified as bot or human, stored as a field rather than dropped, so the stream view can exclude crawlers by default and let the user toggle them back.

**This is the ticket most likely to be underestimated.** Bot detection is not a regex over the user agent — it's accumulated heuristics that mature projects have spent years refining, covering headless browsers, prefetchers, uptime monitors, link previewers, and crawlers that actively disguise themselves. GoatCounter gets this right for free; building it yourself means either pulling in an existing classifier or accepting worse accuracy than the vendor you replaced.

It matters more here than for most projects. At single-digit daily human visits, crawler traffic is the *majority* of the dataset. Get this wrong and every number the tool shows is wrong.

Store the classification, never filter at ingest. A misclassification you can correct later is recoverable; a hit you discarded is gone.

**Blocked by:** 01.

**Status:** ready-for-agent

- [ ] Every hit is stored with a bot classification
- [ ] Known crawlers, uptime monitors and link previewers are classified as bots
- [ ] Ordinary browser traffic is not misclassified as bot
- [ ] Bot hits are stored, never dropped at ingest
- [ ] Classification is a field consumers can filter on, not a hard exclusion
- [ ] Accuracy is compared against a sample of real traffic and the gap versus a mature classifier is written down honestly
