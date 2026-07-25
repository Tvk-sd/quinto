# 04 — `quinto query` — the agent interface

**What to build:** `quinto query "select url_path, count(*) from events group by 1 order by 2 desc"` runs SQL against the local database and prints the result.

This is the project's actual differentiator. Every other terminal dashboard shows a human some numbers. This one lets the agent already running in that terminal interrogate the data directly — no API, no auth, no rate limit, no SDK, and no requirement that a database client be installed.

For that to work, an agent that has never seen this project needs to be able to discover the schema and write a correct query on the first try. That makes schema discoverability part of the deliverable, not documentation to write later.

**Blocked by:** 01 — needs the store. Independent of the TUI; can be built in parallel with 02 and 03.

**Status:** ready-for-agent

- [ ] `quinto query "<sql>"` executes against the local database and prints results
- [ ] An agent can discover the schema through the CLI itself, without reading source
- [ ] Output is readable by a human and parseable by a machine — a structured format is available via a flag
- [ ] Errors in SQL produce a clear message, not a stack trace
- [ ] Works with no database client installed on the machine
- [ ] Read-only — a query cannot modify or drop data
- [ ] Usage is documented in a form an agent will actually encounter (help output, and a snippet suitable for pasting into a project's agent instructions)
