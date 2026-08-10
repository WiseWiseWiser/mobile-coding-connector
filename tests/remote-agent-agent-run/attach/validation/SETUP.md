# Scenario

**Feature**: attach CLI argument and TTY gate validation

```
agent-run attach [bad-args|non-tty] -> Error: clear, non-zero
```

## Preconditions

- Validation runs before a successful live relay.
- Non-interactive path uses L2 `RunWithWriters` (stdout is not a TTY).

## Steps

1. Leaf sets CLI Args for the validation case.
