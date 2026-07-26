# Changelog

All notable changes to quinto are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[semantic versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/Tvk-sd/quinto/releases/tag/v0.1.0
