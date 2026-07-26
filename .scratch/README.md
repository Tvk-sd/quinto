# Working notes

The tickets quinto was built from, and the research behind them. Kept in the
repository on purpose: the reasoning is more transferable than the code, and
most of it is the record of being wrong about something.

```
quinto-v1/issues/     the six tickets v1 was built from, each closed with
                      what actually happened — including the parts that
                      contradicted the plan
quinto-collect/       a second effort, sliced and then deliberately not built:
                      what it would take to run our own collector instead of
                      reading someone else's
tui-polish/           design research for the interface
```

Each ticket is a tracer bullet — a narrow but complete path through every
layer, verifiable on its own — with its blocking edges stated. They were
written before the work and closed afterwards with an `## Answer` section, so
the difference between the plan and the outcome is visible rather than tidied
away.

The most useful reading, if you only want one thing:

- **`quinto-v1/issues/01`** — the sync. Four documented details about the
  GoatCounter export API turned out to be wrong, each of which would have
  produced a parser that passed its own tests and failed on contact.
- **`quinto-v1/issues/02`** — the sample data. Building it *before* the
  interface exposed two modelling errors that six rows of real traffic would
  never have shown.
- **`quinto-collect/`** — the argument for not building something. Slicing the
  work was what settled it: six tickets against v1's six, for a component a
  vendor already provides.

See [PLAN.md](../PLAN.md) for the decisions these came from.
