# Changelog

All notable changes to quinto are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[semantic versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] — 2026-09-26

### Added

- **Multi-site support** — several sites can live in `~/.config/quinto/config`,
  each with its own database. `--site <name>` picks one, and with more than one
  configured the app opens on a start screen listing them with their last sync.
  `quinto sites` prints that list without opening the interface.

### Fixed

- `--db <path>` now works in its documented form. Splitting the subcommand from
  its flags left `--db` behind while its path went into the positional list, so
  the flag failed with "flag needs an argument" — documented, but never usable
  that way.
- The overview's bot label tracks whether bots are currently shown, rather than
  always claiming they are hidden.
- The `/` filter works again over the current overview.
- `sync` retries a 404 on export creation. GoatCounter spends the request
  against the account's budget either way, so treating it as fatal threw away a
  slot that had already been paid for.
- `demo` no longer generates traffic that depends on the day it runs, so the
  sample data is the same shape whenever it is seeded.
- `go.sum` is committed. It had been ignored since the first commit, so a fresh
  clone could not be built without regenerating it.

### Changed

- The export limit is documented as per-account, not per-site — the distinction
  decides how often a multi-site setup can sync.
- README documents `quinto list` and `quinto path`, and the screenshots and demo
  GIF match the current interface.

## [0.1.0] — 2026-07-26

First release.

### Added

- **Stream view** — one row per visit, expandable to the path that visitor took
  through the site, with events shown alongside pageviews.
- **Overview** — visitors, pageviews, events and single-page visits over a
  selectable range, with per-day traffic, top pages, referrers and countries.
- **`/` filter** — matches landing page, referrer, country, browser, and every
  page *inside* a visit, so searching for a page finds visitors who reached it
  second or third rather than only those who arrived on it.
- **`quinto query`** — read-only SQL against the local database, with `--json`
  for machine consumption. `quinto schema` prints the real DDL so an agent can
  write a correct query without reading source.
- **`quinto sync`** — incremental pull from GoatCounter's export API using
  their `last_hit_id` cursor. Rate limiting is reported as a normal state with
  the retry window, not as an error.
- **`quinto demo`** — seeded sample traffic in a separate database, so the tool
  can be tried without an analytics account.
- **`quinto list`** — the same visits as a plain table, for scripts and
  terminals without a TTY.
- Static binaries for macOS and Linux, amd64 and arm64. No cgo, no runtime
  dependencies.

### Notes

Visit durations are `NULL`, and render as `—`, when a visit produced only one
observation: the moment someone leaves is not observable. Bot traffic is stored
rather than discarded, hidden by default, and always counted in the header.

[0.2.0]: https://github.com/Tvk-sd/quinto/releases/tag/v0.2.0
[0.1.0]: https://github.com/Tvk-sd/quinto/releases/tag/v0.1.0
