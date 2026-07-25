# 06 — Ship

**What to build:** Everything that turns a working tool into one a stranger can find, install and understand — ending in a pull request to awesome-tuis under Dashboards, which is this project's definition of done.

The demo GIF carries more weight than any feature. For a terminal tool it is the entire first impression, and it is what people judge before reading a word. Record it against the demo dataset, at realistic density.

Be transparent that the screenshots show seeded data. A portfolio project that implies traffic it doesn't have is the one failure mode that would actually damage the point of building it.

**Blocked by:** 03, 04, 05.

**Status:** ready-for-human — build complete, two items need Till

- [ ] README opens with a demo GIF that makes the tool understandable without reading anything
- [ ] The GIF is recorded against the demo dataset and says so
- [x] Install is a single command or a single binary, on macOS and Linux
- [x] A first-time user gets from install to a populated screen in under a minute
- [x] Setup for real data — GoatCounter account, the individual-pageviews setting, snippet, API token — is documented end to end
- [x] The docs tell users to create an **Export-scoped, single-site token**, and say why — not "create a token" with the permissions left to chance
- [x] The setup docs warn that the "Individual pageviews" toggle must be saved and re-checked after reload, that it is **not retroactive**, and that a populated dashboard is no evidence it is working — every one of these caught us during development
- [x] The agent interface is presented as a feature, not buried in a flags list
- [x] Repository name, binary name and documentation all agree
- [ ] `quinto` name availability confirmed before publishing
- [ ] Pull request opened against awesome-tuis under Dashboards

## Answer

2026-07-25. Everything buildable is done; two items are genuinely not mine.

**Done and verified:**

```
make release
  darwin/arm64  11M     linux/amd64  11M
  darwin/amd64  11M     linux/arm64  10M
```

All four static, `CGO_ENABLED=0` throughout — the payoff from choosing SQLite
over DuckDB back at the start. One machine, no toolchain setup, no cgo.

First run was exercised from an empty `XDG_DATA_HOME`, and every dead end
points somewhere useful:

```
$ quinto
error: no database at …/quinto.db — run `quinto sync` first, or `quinto demo` for sample data

$ quinto sync
error: missing GoatCounter credentials: set site and token in …/config, or export
QUINTO_GOATCOUNTER_SITE and QUINTO_GOATCOUNTER_TOKEN.
Create a token at https://<your-site>.goatcounter.com/user/api with the Export permission only

$ quinto demo
Generated 420 sessions (818 pageviews) over 28 days.
```

Install to populated screen: two commands, well under a minute.

The README documents the GoatCounter setup including all three traps in the
individual-pageviews toggle, specifies the Export-only single-site token and
*why* the scope matters, and gives the agent interface its own section rather
than a line in a flag list. `docs/demo.tape` records the GIF with one command.

**Name check:** the Homebrew formula name `quinto` is free, which is the one
that matters for `brew install`. `github.com/quinto` as an org and `quinto` on
npm are taken, neither of which affects `github.com/<user>/quinto` or a Go
module path.

## Offen — bei Till

1. **Generate the demo GIF.** `brew install vhs && make demo-gif`, then commit
   `docs/demo.gif` and swap the placeholder comment at the top of the README
   for the image link. Left as a comment rather than a link so the README never
   renders a broken image. Not done here because installing vhs pulls ffmpeg
   and ttyd onto your machine — your call, not mine.
2. **Publish and open the PR.** Create the GitHub repo, push, cut a release,
   then open the pull request against awesome-tuis under Dashboards. Both are
   outward-facing actions on your accounts; they belong to you.

Nothing in the code is waiting on either.
