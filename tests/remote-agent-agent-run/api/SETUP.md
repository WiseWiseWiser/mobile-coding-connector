# Scenario

**Feature**: HTTP GET /api/agent-run/sessions

```
seed home -> GET /api/agent-run/sessions[?limit=N] -> {"sessions":[...]}
```

## Preconditions

- API is registered by `agentrun.RegisterAPI(mux, storeHome)`.
- Bearer auth required for `/api/*` (harness uses `lib.TestPassword`).
- Same list semantics as CLI (limit default, sort, DTO fields).

## Steps

1. Leaf sets `Op=api`, optional `APILimit`, and Seeds.
2. Run issues GET; assert status and JSON body.
