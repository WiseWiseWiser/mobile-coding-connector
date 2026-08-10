# Scenario

**Feature**: list with empty agent-run store

```
empty home -> agent-run sessions [--json] -> empty list
```

## Preconditions

- `req.Seeds` is empty (no `sessions/*/meta.json`).
- List still succeeds (exit 0).

## Steps

1. Leave Seeds empty.
2. Leaf selects human vs `--json`.
