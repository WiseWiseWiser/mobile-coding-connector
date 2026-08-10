# Scenario

**Feature**: remote-agent agent-run status (P3)

```
bare | <session-id> | --json -> home path | multi-layer probe | errors
```

## Preconditions

- Bare status uses remote store home (injected RegisterAPI home).
- Per-session status uses ProbeSession inject or library probe.
- No real provider process required for L2 inject leaves.

## Steps

1. Leaf configures CLI Args and optional ProbeInject / Seeds.
