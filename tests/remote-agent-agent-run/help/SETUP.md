# Scenario

**Feature**: agent-run help surfaces (root and sessions)

```
remote-agent agent-run [--help|sessions --help] -> usage text exit 0
```

## Preconditions

- Help is resolved in-process without requiring a seeded store.
- Product wires top-level `agent-run` and `agent-run sessions` help.

## Steps

1. Leaf sets CLI Args for the help target.
2. Assert exit 0 and expected usage keywords.

## Context

P1 requires at least `sessions` listed under `agent-run` help and list flags on
`sessions --help`.
