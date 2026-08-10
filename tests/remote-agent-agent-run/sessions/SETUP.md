# Scenario

**Feature**: CLI `agent-run sessions` list mode

```
seed store home -> remote-agent agent-run sessions [--json] [--limit N]
  -> human table | JSON {sessions:[...]}
```

## Preconditions

- CLI talks to remote `/api/agent-run/sessions` (server holds the store home).
- Empty vs seeded store is controlled only by `req.Seeds`.

## Steps

1. Leaf chooses empty or seeded store and human vs JSON / limit flags.
2. Run CLI; assert columns, counts, or JSON payload.

## Context

Matches local `agent-run sessions` list UX as closely as practical for P1.
