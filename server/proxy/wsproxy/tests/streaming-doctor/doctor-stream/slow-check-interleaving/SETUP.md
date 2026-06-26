# Scenario

**Feature**: slow upstream_fetch does not block fast checks

```
# test hook delays upstream_fetch 200ms; earlier checks emit immediately
config_load/upstream_proxy -> SSE (t0)
upstream_fetch (delayed 200ms) -> SSE (t0+200ms)
```

## Preconditions

`SetTestUpstreamFetchDelay(200ms)` active via test hook.

## Steps

1. Set `UpstreamFetchDelayMs = 200`.

## Context

Requirement scenario `slow-check-interleaving`. Uses recorded arrival order of
progress ids as a proxy for timestamps (parsed sequentially from SSE body).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UpstreamFetchDelayMs = 200
	return nil
}
```
