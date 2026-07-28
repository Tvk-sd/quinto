# VHS tape for the start-screen (ticket 02) GIF — not wired into `make
# demo-gif`; run directly with `vhs docs/start-screen.tape`.
#
# Recorded against a throwaway fixture (.scratch/quinto-v2/giffixtures/),
# never against real config — "blog" and "shop" are fabricated names with
# synthetic demo-shaped data, not a real account. The env exports below are
# hidden from the recording the same way demo.tape hides its `clear`.

Output docs/start-screen.gif

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

Type "./quinto"
Sleep 400ms
Enter
Sleep 3s

# Walk the list — three rows, three different freshness states.
Down@400ms 2
Sleep 1500ms

# Open the top row's site.
Up@400ms 2
Sleep 600ms
Enter
Sleep 3s

# Back to the list — q here isn't a new keybinding, it's today's existing
# quit, which is exactly what makes this "back" rather than new behaviour.
Type "q"
Sleep 2s

Type "q"
Sleep 1s
