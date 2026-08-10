# Scenario

**Feature**: list with seeded agent-run session metas

```
home/sessions/<id>/meta.json × N -> agent-run sessions [flags]
  -> limited / ordered list
```

## Preconditions

- Leaves set `req.Seeds` to one or more metas with known ids and timestamps.
- Default list limit is 10 (local agent-run parity).

## Steps

1. Seed store via root Run.
2. Invoke CLI with limit/json flags as specified by the leaf.
