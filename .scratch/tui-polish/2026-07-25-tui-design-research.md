# TUI design + QoL research — what to steal from pi, hermes, and the Charm v2 stack

Research note, 2026-07-25. Ideation only — not a roadmap, not ranked, nothing picked.

---

## BLUF

pi and hermes are **agent-chat TUIs**: a scrolling transcript plus a composer, driven by a
streaming turn lifecycle. quinto is a **navigational dashboard**: read-only, keyboard-driven,
no text input. Their *craft* transfers (differential rendering, themes, overlays, remappable
keys, command palette); their *layout* does not (composer, streaming pane, queue/interrupt/abort).

The single highest-leverage item below is infrastructure, not a feature: **Charm shipped
Bubble Tea / Lip Gloss / Bubbles v2 on 2026-02-23**, and quinto is on v1. That upgrade buys
most of what makes pi and hermes feel good, for free.

Also: pi is TypeScript, hermes is Python+TypeScript. **Patterns transfer, code does not.**
Nothing below is drop-in.

---

## How they are built

### pi — `earendil-works/pi` (MIT, TypeScript, ~73k★)

Two layers worth studying: `@earendil-works/pi-tui` (the framework) and the coding agent on top.

**pi-tui rendering.** Custom framework, no React. `Component` interface is a single
`render(width: number): string[]` — components return an array of lines and cache them keyed
on width. The TUI then picks one of **three rendering strategies**:

1. *First render* — output all lines, do not clear scrollback
2. *Width changed, or a change above the viewport* — clear screen, full re-render
3. *Normal update* — move cursor to first changed line, clear to end, render only changed lines

Every update is wrapped in **synchronized output** (`\x1b[?2026h` … `\x1b[?2026l`) so the
terminal paints atomically. That is the anti-flicker mechanism, and it is exactly what
Bubble Tea v2 now does by default.

**Overlays.** `tui.showOverlay(component, opts)` draws on top of existing content without
replacing it — anchor-based (`center`, `bottom-right`, …) or percentage positioning, with
`width` / `minWidth` / `maxHeight` accepting either absolute columns or `"80%"`. Used for
dialogs, pickers, menus.

**Theming.** Components accept a theme *interface* of style functions
(`borderColor: (str) => string`, nested `selectList` theme, …) rather than reading globals.
Styling is injected, so a theme swap is a data change, not a code change.

**Keybindings.** Every action has a namespaced id (`tui.editor.cursorUp`,
`tui.input.submit`, …), all remappable in `~/.pi/agent/keybindings.json`, multiple keys per
action, `/reload` applies changes live without restarting, and pre-namespaced legacy ids are
auto-migrated on startup. `/hotkeys` lists everything.

**Discoverability.** `/` opens slash-command completion with descriptions; `@` fuzzy-searches
project files; Tab completes paths. Autocomplete is a pluggable provider
(`CombinedAutocompleteProvider`), not hardcoded.

Other bits: bracketed-paste with a `[paste #1 +50 lines]` collapse marker for >10-line pastes;
inline images via Kitty/iTerm2 graphics protocols; fake cursor rendering (real cursor hidden).

### hermes — `NousResearch/hermes-agent` (Python core + `ui-tui/`)

**Split-process architecture.** `ui-tui/` is **React + Ink** (TypeScript). It owns the screen
and nothing else. Python owns sessions, tools, model calls, and command logic. The TS client
spawns `python -m tui_gateway.entry` and talks **newline-delimited JSON-RPC over stdio**:

```
ui-tui/src                  tui_gateway/
entry.tsx  →  GatewayClient  ⇄  entry.py → server.py RPC handlers
           →  App
stdin/stdout: JSON-RPC requests, responses, events
stderr:       captured into an in-memory log ring
```

Malformed stdout is surfaced as `gateway.protocol_error`; stderr becomes `gateway.stderr`.
**Neither ever writes to the terminal** — that discipline is why the UI never gets corrupted
by a stray print. Worth copying regardless of language.

**State.** nanostores (`turnStore`, `overlayStore`, `uiStore`, `delegationStore`) rather than
one god-object. Rendering uses Ink's `Static` for settled transcript rows plus live rows above
the input, so finished output is never re-rendered.

**The UX list they advertise** (this is the actual QoL menu):

- **Instant first frame** — banner paints before the app finishes loading; terminal never looks frozen
- **Non-blocking input** — you can type and queue before the session is ready
- **Rich overlays** — model picker, session picker, approvals render as modal panels, not inline flows
- **Live panel** — tools/skills fill in progressively as they initialize
- **Mouse-friendly selection** — drag highlights with a uniform background instead of SGR inverse, so the terminal's native copy gesture works
- **Alternate-screen + differential updates** — no flicker while streaming, no scrollback clutter after quit
- **Live theme swap** — `/skin ares` repaints the whole UI mid-session; skins are a keyed palette (banner, UI colors, prompt glyph, completion menu, selection bg, `tool_prefix`, `help_header`)
- Slash autocompletion opens as a **floating panel with descriptions**, not an inline dropdown

### Charm v2 — what quinto is currently missing

quinto: `bubbletea v1.3.10`, `lipgloss v1.1.0`. Released 2026-02-23/24:

- **Cursed Renderer** — rebuilt on the ncurses rendering algorithm, heavily optimised. Free on upgrade.
- **Mode 2026 synchronized output** — on by default, kills tearing/cursor flicker. This is pi's hand-rolled trick, built in.
- **Progressive keyboard enhancements** (Kitty protocol) — `shift+enter`, `ctrl+h`, key-release events become bindable; `tea.KeyboardEnhancementsMsg` tells you if the terminal supports it, so you can fall back.
- **Color profile as a message** — `tea.ColorProfileMsg`, `tea.WithColorProfile(…)` for forcing TrueColor/ANSI256/ANSI/Ascii/NoTTY. Makes screenshots and the demo tape reproducible.
- **DECRPM mode reports** — query whether the terminal supports focus events, sync output, etc.
- **`tea.SetClipboard` / `tea.ReadClipboard`** — OSC 52, pure Go, works over SSH. Verified
  *absent* from v1.3.10 (no `Clipboard` symbol anywhere in the module), so this one genuinely
  requires the upgrade — or a hand-rolled OSC 52 write, see item 9.
- **Declarative `tea.View`** instead of returning a bare string. `tea.WithAltScreen()` and
  friends are gone; you set `v.AltScreen = true` in `View()` instead.

**Migration cost, checked against the official `UPGRADE_GUIDE_V2.md` and this codebase**
(~700 LOC across `tui.go`, `overview.go`, `format.go`). The guide's checklist is 12 items;
most are N/A here because quinto uses no mouse and none of the removed commands:

| Guide item | Where it lands in quinto | Effort |
|---|---|---|
| Import paths → `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2` | all 3 TUI files | trivial |
| `View() string` → `View() tea.View` | `tui.go:218` — one method, wrap the return | small |
| `tea.KeyMsg` → `tea.KeyPressMsg` | `tui.go:137` — one `case` | trivial |
| **`case " "` → `case "space"`** | `tui.go:159` (`case "enter", " ", "right", "l"`) | trivial, easy to miss |
| `tea.WithAltScreen()` removed → `v.AltScreen = true` | `tui.go:451` | trivial |
| `Update(msg) (tea.Model, tea.Cmd)` | **unchanged in v2** — no work | — |
| Mouse messages, removed commands/methods, `WindowSize()`, `Sequentially()` | not used | — |
| **`lipgloss.AdaptiveColor` removed** (Lip Gloss v2) — `tui.go:41–49` and the styles above it | `compat` package restores it, or make light/dark explicit | small |
| Re-verify the VHS demo tape output | — | small but must be checked |

So: a focused half-day, and the estimate now rests on the guide rather than a guess. The only
open judgement call is the AdaptiveColor removal — `compat` makes it a one-line fix, doing it
properly means deciding light/dark explicitly, which overlaps with item 2 anyway.

**Unrelated nit:** `bubbletea` and `lipgloss` are marked `// indirect` in `go.mod` despite
`internal/tui` importing both directly — `go mod tidy` hasn't run since they were added.

---

## Feature menu

Grouped, not ranked. "Cost" is effort in *this* codebase as it stands today.

### Render & feel

**1. Upgrade to Charm v2.** — ✅ **shipped 2026-07-25**
Cursed Renderer + Mode 2026 + Kitty keyboard + OSC 52 clipboard, all at once. Everything in
this section and half of the next gets cheaper afterwards.
*Landed on `bubbletea v2.0.8` / `lipgloss v2.0.5`. Actual cost was well under the half-day
estimate — the five touchpoints in the table above were the whole job, plus porting the two
test files off `tea.KeyMsg{Type:, Runes:}`. `View()` became a thin `tea.View` wrapper around a
new unexported `render() string`, which kept every existing test assertion intact.
Verified: `go vet` clean, full suite green, all four release targets cross-compile with
`CGO_ENABLED=0`. Not verified: what it looks like — see PLAN.md › Offen.*

**2. A theme file.**
Colours are hardcoded lipgloss styles at `tui.go:30–50`. pi injects a theme interface;
hermes ships named skins swappable live. Minimum viable version: a `[theme]` block in
`~/.config/quinto/config` mapping named roles (`accent`, `dim`, `bot`, `error`, `cursor`) to
colours, loaded in `New()`. Live-swap is a stretch goal, a config key is 80% of the value.
*Cost: small — one struct, one loader, replace package-level vars.*

**3. Async load with a real loading state.**
`reload()` (`tui.go:104`) runs synchronously inside `Update`. Fine at today's data volume —
the comment at `tui.go:87` says so explicitly and is correct — but it means any future slow
query freezes input. hermes' "instant first frame" is the same idea: paint the chrome, fill
in the data. Move `reload()` into a `tea.Cmd`, add a spinner only for the >100ms case.
*Cost: medium — touches Model lifecycle, is the kind of change that's cheaper now than later.*

### Navigation & discoverability

**4. Fuzzy filter (`/`).**
The k9s/lazygit convention: `/` filters the visible list live — by path, referrer, country,
browser. Currently the only filter is `b` (bots). This is the single biggest gap between
quinto and a dashboard TUI people call "good".
*Cost: medium — needs a text input, a filter predicate, and a decision about whether filtering
happens in SQL or in Go over the loaded slice. **Adds a direct dep on `bubbles`** (`textinput`);
today the TUI is entirely hand-rolled and `bubbles` is not in `go.mod` at all. Pure Go, so no
cgo problem, but it is a real trade against "simplicity first" — hand-rolling a single-line
input is maybe 60 lines.*

**5. Command palette (`:`).**
Everything reachable by name with descriptions: `:range 30d`, `:bots on`, `:export csv`,
`:copy query`. pi's `/` menu and hermes' floating completion panel are the reference. Pays off
most once there are more than ~10 actions — arguably premature today, but it's the pattern
that makes adding action #11 free.
*Cost: medium — reuses whatever item 4 builds.*

**6. Searchable, overlay help.**
`?` currently replaces the entire screen (`tui.go:425`). hermes and pi render help as an
overlay over live content, and lazygit has an open request for a *filterable* keybinding menu.
Adopting `bubbles/key` + `bubbles/help` would make the keymap a data structure — which then
feeds the footer hints, the help screen, and item 7 from one source.
*Cost: medium. **Adds a direct dep on `bubbles`** (`key` + `help`) — same trade as item 4.
Note the footer at `tui.go:392` already degrades hints gracefully by width — that's good work
worth preserving, and a keymap struct makes it declarative rather than a hardcoded ladder.*

**7. Remappable keys.**
pi's model: namespaced ids, multiple bindings per action, live `/reload`. Overkill at quinto's
size unless item 6 lands first — after that it's mostly serialising the keymap struct.
*Cost: small if 6 exists, medium otherwise.*

**8. A range picker instead of `r`-cycles.**
`r` cycles forward through `Range` (`overview.go:44`) with no way back. An overlay picker
(hermes renders every picker as a modal panel) or `shift+r` to cycle backwards.
*Cost: trivial for reverse-cycle, small for the overlay.*

### Data interaction

**9. Copy to clipboard.**
`tea.SetClipboard` is OSC 52 — pure Go, no cgo, works over SSH. Copy the selected session's
URL, its id, or the whole visible table as TSV. pi binds `ctrl+x` to "copy last response";
this is the same reflex.
*Cost: trivial. **Unblocked** — item 1 shipped 2026-07-25, so `tea.SetClipboard` is available
now. (It was genuinely absent from v1.3.10; there was no `Clipboard` symbol in that module.)*

**10. "Show the query" — the agent-parity feature.**
Press a key, see the exact `quinto query "…"` invocation behind the current view; copy it via
item 9. This is the one idea on this list that is *specific to quinto* rather than borrowed:
CLAUDE.md frames the project as "the same file is queryable by agents", and this makes the TUI
teach the CLI. It also enforces a useful discipline — every new view must be expressible as
one SQL string.
*Cost: small-medium; the constraint is real design work, not code.*

**11. Drill-down from Overview.**
In `overviewView` the columns (`overview.go:173`) are static lists. Select a path/referrer/
country → jump to the stream filtered to it. Turns two screens into one navigable model.
*Cost: medium — needs cursor state per column and a filter to hand to the stream, so it
naturally follows item 4.*

**12. Auto-refresh / live tail.**
A `tea.Tick` re-running the sync-state check, with an unobtrusive "3 new visits — press R"
indicator rather than yanking the cursor. Note `r` is taken by range; needs a key decision.
*Cost: small once item 3 (async load) exists; awkward before it.*

---

## Deliberately not worth it for quinto

- **Chat composer / streaming pane / turn lifecycle** (queue, steer, interrupt, abort). The
  whole reason pi and hermes look the way they do. quinto has no text input and nothing streams.
- **Split-process JSON-RPC architecture.** hermes needs it because the UI is TS and the core is
  Python. quinto is one Go binary — that's the design win, don't spend it.
- **Extension/plugin system, RPC mode, slash-command registry for third parties.** pi's
  extension surface exists because it's a platform. quinto is a tool.
- **Inline images (Kitty/iTerm2 graphics).** Tempting for the sparkline. A braille-resolution
  sparkline in plain text degrades everywhere; a graphics-protocol chart degrades to nothing.
- **Mouse-first interaction.** hermes' mouse work is specifically to make *text selection*
  behave; adopt that narrow goal if anything, not clickable UI.
- **A web dashboard.** GoatCounter already is one. The point of quinto is that it isn't.

## Constraints any of the above must respect

- **No cgo** (CLAUDE.md, hard). Everything named here is pure Go: Charm v2 stack, OSC 52
  clipboard, bubbles. No native clipboard library (`golang-design/clipboard` et al. is cgo on
  some platforms) — OSC 52 is the correct answer anyway.
- **`quinto query` parity.** Items 4, 8, 11 all add filters or views. Each should be expressible
  as a `quinto query` invocation, or the "same file queryable by agents" framing quietly erodes.
  Item 10 makes that check automatic.

---

## Sources

- [earendil-works/pi (pi-mono)](https://github.com/badlogic/pi-mono) · [pi-tui README](https://raw.githubusercontent.com/badlogic/pi-mono/main/packages/tui/README.md) · [usage docs](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/usage.md) · [keybindings](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/keybindings.md)
- [NousResearch/hermes-agent](https://github.com/nousresearch/hermes-agent) · [ui-tui implementation README](https://github.com/NousResearch/hermes-agent/tree/main/ui-tui) · [TUI user guide](https://hermes-agent.nousresearch.com/docs/user-guide/tui)
- [Bubble Tea v2.0.0 release notes](https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0) · [Upgrade guide](https://github.com/charmbracelet/bubbletea/blob/v2.0.0/UPGRADE_GUIDE_V2.md) · [Lip Gloss v2: What's New](https://github.com/charmbracelet/lipgloss/discussions/506) · [Bubbles v2.0.0](https://github.com/charmbracelet/bubbles/releases/tag/v2.0.0)
- [Bubble Tea clipboard (OSC 52)](https://github.com/charmbracelet/bubbletea/blob/main/clipboard.go) · [go-osc52](https://github.com/aymanbagabas/go-osc52)
- [lazygit #4846 — filterable keybinding menu](https://github.com/jesseduffield/lazygit/issues/4846) · [Command Palette pattern](https://uxpatterns.dev/patterns/advanced/command-palette)
