# VHS tape for the README GIF — `make demo-gif` (needs: brew install vhs).
#
# Recorded against the demo dataset on purpose. Real traffic on a small site
# is a handful of rows, which would undersell the tool and tell a viewer
# nothing about how it behaves with data.
#
# Two mistakes produced an empty first recording; both are easy to reintroduce.
# The binary must be invoked as ./quinto, because vhs starts a plain shell with
# no PATH entry for it. And Tab is a VHS command — Type "\t" sends a literal
# backslash-t. When either is wrong the keystrokes land in the shell instead of
# the app, and the GIF records nothing but a prompt.

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
Show

Type "./quinto --demo"
Sleep 400ms
Enter
Sleep 3s

# Walk down to a visit worth opening.
Down@250ms 6
Sleep 900ms

# Expand it: the journey is the whole point of the tool.
Enter
Sleep 4s

Down@250ms 3
Sleep 1s
Enter
Sleep 3s

# The overview.
Tab
Sleep 4s

# Widen the range.
Type "r"
Sleep 3s

Type "q"
Sleep 1s
