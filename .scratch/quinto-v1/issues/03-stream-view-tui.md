# 03 — Stream view TUI

**What to build:** Running `quinto` opens a full-screen terminal view of recent traffic, grouped by session and expandable to reveal the path each visitor took through the site.

This is the core screen and the reason the project exists. It reads the local database directly and never touches the network.

Shape decided during ticket breakdown — one row per visit, expandable into the journey:

```
  ▼ 14:22  DE · Chrome · google.com     3 pages · 2m14s
      14:22  /
      14:23  /pricing
      14:24  /signup

  ▶ 14:19  US · Safari · direct           1 page  · 8s

  ▶ 14:03  GB · Firefox · news.yc.com     5 pages · 6m02s
```

**Why session-grouped and not a flat list:** at low traffic the only real story in the data is what a single visitor did. Aggregates need volume that this tool's users mostly won't have. This is the honest, small-n form of a customer journey map — the original ask, rendered in the only way that isn't a lie at n=1.

Build it against the demo dataset so it's designed for realistic density, then verify against real synced data.

**Blocked by:** 01, 02.

**Status:** ready-for-agent

- [ ] `quinto` opens a full-screen view listing recent sessions, newest first
- [ ] Each session row shows time, geography, browser, referrer, page count and duration
- [ ] A session expands and collapses to show the ordered path of pages visited
- [ ] Keyboard navigation moves between sessions; expand/collapse and quit are discoverable without documentation
- [ ] The view adapts to terminal width and height, including small windows
- [ ] It renders correctly with hundreds of sessions and with a nearly empty database
- [ ] An empty database shows a message explaining how to get data, not a blank screen
- [ ] Bots are excluded by default and can be toggled back into view
- [ ] Opening the view performs no network requests
