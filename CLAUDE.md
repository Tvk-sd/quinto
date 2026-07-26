# quinto

Terminal-native web analytics. Reads a local SQLite file synced from
GoatCounter; the same file is queryable via `quinto query`.

## Constraints worth knowing before changing anything

**No cgo.** Storage is `modernc.org/sqlite`, pure Go, so the project
cross-compiles to a single static binary for macOS and Linux on both
architectures from one machine. A cgo dependency breaks that, and with it the
install story.

**The schema is a public interface.** `hits` and the `sessions` view are what
an agent reads through `quinto query`. Column names favour legibility over
brevity and mirror GoatCounter's export rather than inventing a vocabulary.
Renaming one is a breaking change, not a refactor.

**The interface never talks to the network.** Only `sync` does. That boundary
is what makes the UI instant, usable offline, and readable by anything that can
open a file.

## Rules the code follows deliberately

Each of these exists because the alternative asserts something the data does
not support:

- **Unknown is not zero.** A visit with one observation has `NULL` duration,
  never `0` — you cannot see when someone left. `AVG` skipping NULL is
  intended, not a bug to work around.
- **Unknown is not "direct".** A referrer that was never synced renders `?`;
  real direct traffic renders `direct`.
- **Bots are stored, never dropped at ingest.** They are filtered in queries
  and hidden in the UI with the count always shown. A filter can be lifted; a
  discarded row cannot be recovered.
- **Empty days draw nothing** in the traffic chart, and fewer than five days of
  data is labelled as too sparse to call a trend.

## Testing

Test the sync logic, not the rendering. Cursor handling, idempotency and
export-format version checks cannot be verified by eye. The GoatCounter client
is tested against a real captured export in `internal/goatcounter/testdata/` —
the published docs were wrong in four places, so fixtures written from
documentation would encode those errors.

`make test` before pushing. `make demo-gif` re-records the README GIF after any
UI change: the tape walks a fixed path, so changing which screen opens first
silently turns the GIF into a picture of software that no longer exists.
