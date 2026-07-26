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
#
# The order below follows the app, and the app changed: quinto now opens on the
# overview, so this starts there and tabs *into* the stream. The previous tape
# did the reverse and would now record the two screens back to front.

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

# Opens on the overview: what happened, before who came.
Sleep 3s

# Widen the range — the sparkline and the numbers move together.
Type "r"
Sleep 3s

# Into the stream: one row per visit.
Tab
Sleep 2s

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

# Filter. "contact" is the demo worth showing: nobody lands on /contact,
# they reach it second or third — so the list shrinks hard and every row
# has to say which page matched, because none of them displays the word.
Type "/"
Sleep 700ms
Type "contact"
Sleep 4s

Escape
Sleep 1500ms

Type "q"
Sleep 1s
