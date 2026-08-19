# Scenario

**Feature**: native-terminals is local-agent only; aliases equivalent; bare terminals gone

```
remote-agent native-terminals|native-terminal|native-terms|native-term
  -> unknown command: <typed>
local-agent native-term foo
  -> Error: unknown native-terminals subcommand: foo
each alias + list -h -> Usage native-terminals + --json
local-agent terminals list -> unknown command: terminals
```
