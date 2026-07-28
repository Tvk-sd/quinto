# VHS tape for the README GIF — `make demo-gif` (needs: brew install vhs).
#
# Recorded against a throwaway fixture (`make giffixture`), not real config:
# several fabricated sites ("blog", "shop"), never a real account, so the
# recording shows the start screen honestly without exposing any real
# domain or traffic. The env exports below are typed and hidden the same
# way the initial `clear` is — they point XDG_CONFIG_HOME/XDG_DATA_HOME at
# the fixture rather than a real config.
#
# Two mistakes produced an empty first recording; both are easy to
# reintroduce. The binary must be invoked as ./quinto, because vhs starts a
# plain shell with no PATH entry for it. And Tab is a VHS command — Type
# "\t" sends a literal backslash-t. When either is wrong the keystrokes land
# in the shell instead of the app, and the GIF records nothing but a prompt.
#
# The walkthrough now starts at the multi-site start screen (ticket 02),
# opens a site into the overview, widens the range, toggles bots to show
# the exclusion is visible and reversible, moves to the stream, expands a
# visit, filters, then goes back to the start screen the same way the
# dashboard always quits — no new keybinding, just today's q/esc.

Output docs/demo.gif

Set FontSize 15
Set Width 1100
Set Height 640
Set Padding 18
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms

Hide
Type "clear"
Enter
Type "export XDG_CONFIG_HOME=.scratch/quinto-v2/giffixtures/config"
Enter
Type "export XDG_DATA_HOME=.scratch/quinto-v2/giffixtures/data"
Enter
Type "clear"
Enter
Show

# Start screen: several sites, at a glance which is worth opening.
Type "./quinto"
Sleep 400ms
Enter
Sleep 2500ms

# Open the top site — straight into the overview, not a raw visit list.
Enter
Sleep 2500ms

# Widen the range — the sparkline and the numbers move together.
Type "r"
Sleep 2s

# Bots: hidden by default but disclosed, and the exclusion is reversible.
Type "b"
Sleep 1500ms
Type "b"
Sleep 1s

# Into the stream: one row per visit.
Tab
Sleep 1500ms

Down@250ms 5
Sleep 900ms
Enter
Sleep 3s

# Filter. "contact" is the demo worth showing: nobody lands on /contact,
# they reach it second or third — so the list shrinks hard and every row
# has to say which page matched, because none of them displays the word.
Type "/"
Sleep 700ms
Type "contact"
Sleep 3500ms
Escape
Sleep 1s

# Back to the start screen — today's existing quit, not a new keybinding.
Type "q"
Sleep 2s

Type "q"
Sleep 1s
