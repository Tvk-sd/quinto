# The dashboard-TUI genre — how six of them are built

Survey, 2026-07-25. Companion to `2026-07-25-tui-design-research.md`, which covered the
agent-chat TUIs (pi, hermes). This one covers the genre quinto actually belongs to.

---

## BLUF

Six tools, six different frameworks, and a strikingly uniform set of conventions on top.
The genre has converged on four things — **keybindings as user-editable data that also
generates the help menu**, **themes as named files in an XDG directory**, **`/` to filter**,
and **fuzzy matching as a dependency rather than something you write**.

The most directly useful finding: **gh-dash is on `charm.land/bubbletea/v2 v2.0.2`** — the
same stack quinto moved to this afternoon, two patch versions behind. The upgrade wasn't
speculative; it's where the genre already is.

---

## Comparison

| Tool | Language | Framework | Navigation model | Keymap story | Theming |
|---|---|---|---|---|---|
| **k9s** | Go | `derailed/tview` + `derailed/tcell/v2` (both forks) | Resource-centric. `:` command mode with aliases (`:pod`, `:dp`, `:xray`), breadcrumbs, `<esc>` unwinds | `hotkeys.yaml`, hot-reloaded, custom keys appear in `?` | YAML skins in `$XDG_CONFIG_HOME/k9s/skins`, per-cluster, `K9S_SKIN` env override |
| **lazygit** | Go | `pkg/gocui` — gocui **vendored into the repo**, on `gdamore/tcell/v3` | Fixed panels, numbered jumps, `?` menu, popups for everything modal | `config.yml`, JSON schema published for editor IntelliSense | `config.yml` theme block; author/branch colour *patterns* |
| **btop** | C++23 | None — hand-rolled terminal drawing | "Game-inspired menu system", full mouse, every highlighted key clickable | Help menu; UI menu edits all config options in-app | `.theme` files in XDG dir, **bpytop/bashtop-compatible format** |
| **gh-dash** | Go | `charm.land/bubbletea/v2` + `bubbles/v2` + `lipgloss/v2` + `bubblezone/v2` | Config-defined sections, prev/next section, preview pane toggle | YAML `{key, builtin\|command, name}` — `name` is what shows in the help menu | Theme block in the same layered YAML config (`koanf`) |
| **posting** | Python | Textual | GUI-like. `ctrl+o` jump mode (overlay letters, QWERTY-positioned), tab focus, full mouse | Configurable keymap; `f1` gives **contextual** help for the focused widget | User-defined theme files |
| **ATAC** | Rust | `ratatui` 0.30 + a widget ecosystem (tree, scrollview, throbber, image) | Postman-like panes, vim keybindings | `crokey` crate parses user keybinding files | Theme files |

---

## Conventions, counted

Only listing what shows up in **three or more** of the six — at this sample size, two is a
coincidence.

**1. Keybindings are data, the help menu is generated from it. (6/6.)**
Every single one. k9s reloads `hotkeys.yaml` live and lists your custom keys in `?`.
gh-dash's config entry carries a `name` field whose only job is to describe the binding in
the help menu. lazygit publishes a JSON schema so your editor autocompletes the config.
ATAC pulled in a crate (`crokey`) just to parse keybinding files.
The insight isn't "let users remap keys" — it's that **one data structure feeds both the
handler and the help**, so they can't drift.

**2. Themes are named files in an XDG directory, not code. (6/6.)**
k9s: `$XDG_CONFIG_HOME/k9s/skins/*.yaml`, selectable per cluster or via `K9S_SKIN`.
btop deliberately kept the **bpytop/bashtop theme format** so it inherited an existing theme
library on day one. Both are worth noting: theming is a distribution strategy, not just a
preference — named themes get shared, screenshotted, and blogged.

**3. `/` filters. (4/6.)**
k9s goes furthest: `/foo` regex-filters, `/!foo` inverts, `/-f foo` switches to fuzzy,
`/-l app=x` filters by label. lazygit, btop and gh-dash all have a search/filter on `/` or a
dedicated `search` builtin. Note `:` for command mode is only k9s and lazygit — **not** a
convention at this sample size, whatever the vim reflex says.

**4. Fuzzy matching is a dependency. (3/3 of the Go tools.)**
`sahilm/fuzzy` appears in k9s, lazygit *and* gh-dash. Nobody writes their own matcher.

**5. Clipboard via `atotto/clipboard`. (3/3 of the Go tools.)**
Same three. It shells out to `pbcopy`/`xclip` — no cgo, but it needs a local clipboard to
shell out to. quinto now has `tea.SetClipboard` (OSC 52) from the v2 upgrade, which is the
better choice for a tool people run over SSH, since the escape sequence reaches the *client's*
clipboard rather than the server's.

**6. An escape hatch to the underlying CLI. (3/6.)**
gh-dash and lazygit both let you bind a key to a templated shell command
(`gh pr merge --repo {{.RepoName}} {{.PrNumber}}`); k9s has plugins. This is how the genre
gets extensibility without a plugin API.

**Worth flagging:** this is the strongest idea in the survey *and* the one quinto should
adopt in its weakest form. Shelling out to user-configured commands turns a read-only
dashboard into an arbitrary-execution surface, with the support burden that implies. The
version already on quinto's list — "show the query", revealing and copying the
`quinto query "…"` behind the current view — gets most of the value with none of that.
Present the shell-out as what the genre does, not as the target.

---

## What quinto already does right

The survey validates work that's already in the code:

- **The responsive footer ladder** (`tui.go:392`) — five progressively shorter hint strings,
  picking the longest that fits. Every tool here puts key hints in a footer; quinto's
  degradation is more careful than most, which typically just truncate.
- **The sparkline's refusal to draw empty days** (`overview.go:118`) — zero-traffic days
  render as a space, never a baseline block, and under five days of data it says
  *"too few days to call it a trend"*. Nothing in this survey does anything comparable.
  That's a genuine differentiator, not a gap.
- **Bots as a liftable filter rather than deleted rows**, with the count surfaced in the
  empty state. Same instinct as k9s's `/!` inverse filter: filtering is a view, not a delete.
- Vim keys *and* arrows bound together; `?` for help; alt-screen. All genre-standard.

Measured against the conventions above, the actual gaps are: **no theme file** (6/6 have one),
**keymap is not data** (6/6 have it), **no `/` filter** (4/6), **no clipboard** (3/3 of the Go
tools). Those four are the shortlist.

---

## One correction to the earlier note

In `2026-07-25-tui-design-research.md`, item 4 (fuzzy filter) was costed as *"medium — adds a
`bubbles` dep, or hand-roll a single-line input in ~60 lines"*. That mis-framed it. The genre
answer for the matching half is `sahilm/fuzzy` — a small pure-Go library that all three Go
tools in this survey use. Only the *input widget* is a build-or-borrow decision; the matcher
is not. Item 4 is cheaper than I wrote.

---

## Sources

- [k9s](https://github.com/derailed/k9s) · [go.mod](https://raw.githubusercontent.com/derailed/k9s/master/go.mod) · [commands & key bindings](https://k9scli.io/topics/commands/) · [hotkeys](https://k9scli.io/topics/hotkeys/) · [skins](https://k9scli.io/topics/skins/)
- [lazygit](https://github.com/jesseduffield/lazygit) · [go.mod](https://raw.githubusercontent.com/jesseduffield/lazygit/master/go.mod) · [pkg/gui/gui.go (vendored gocui import)](https://github.com/jesseduffield/lazygit/blob/master/pkg/gui/gui.go) · [Config.md](https://github.com/jesseduffield/lazygit/blob/master/docs/Config.md) · [releases (v0.58 tcell v3 notes)](https://github.com/jesseduffield/lazygit/releases)
- [btop](https://github.com/aristocratos/btop) — README: features, themes, help menu
- [gh-dash](https://github.com/dlvhdr/gh-dash) · [go.mod](https://raw.githubusercontent.com/dlvhdr/gh-dash/main/go.mod) · [custom keybindings](https://gh-dash.dev/configuration/keybindings/)
- [posting](https://github.com/darrenburns/posting) · [navigation guide](https://posting.sh/guide/navigation/)
- [ATAC](https://github.com/Julien-cpsn/ATAC) · [Cargo.toml](https://raw.githubusercontent.com/Julien-cpsn/ATAC/main/Cargo.toml)
