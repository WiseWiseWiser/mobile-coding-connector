# Scenario

**Feature**: inline split TUI (library View/ApplyKey + CLI --tty)

```
# library
NewUIState(last-good) -> View lines (Terminals chrome, sidebar, list, cached)
ApplyKey(j|tab|]|enter|q) -> state / focus|quit action

# CLI
native-terminals list --tty + SetTerminalsKeys -> paint then q / enter focus
```

## Context

`Op=tui` leaves call `itermswitcher` only (no RunWithWriters).
`tui/cli/` leaves use CLI with `--tty` and `Keys` via testhooks.
