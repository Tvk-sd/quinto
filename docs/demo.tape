# VHS tape for the README GIF — `make demo-gif` (needs: brew install vhs).
#
# Recorded against the demo dataset on purpose. Real traffic on a small site
# is a handful of rows, which would undersell the tool and tell a viewer
# nothing about how it behaves with data.

Output docs/demo.gif

Set FontSize 15
Set Width 1100
Set Height 640
Set Padding 18
Set Theme "Catppuccin Mocha"
Set TypingSpeed 55ms

Type "quinto --demo"
Sleep 400ms
Enter
Sleep 2.5s

# Walk down to a visit worth opening.
Down@250ms 6
Sleep 800ms

# Expand it: this is the whole point of the tool.
Enter
Sleep 3.5s

Down@250ms 3
Sleep 1.2s
Enter
Sleep 3s

# The overview.
Type "\t"
Sleep 3.5s

# Widen the range.
Type "r"
Sleep 2.5s

Type "q"
Sleep 800ms
