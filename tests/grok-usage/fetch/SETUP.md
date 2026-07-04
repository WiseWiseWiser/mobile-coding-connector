# Scenario

**Feature**: grok usage service fetch via mock binary

```
GROK_SHOW_USAGE_BIN mock -> service FetchOnce -> GrokUsageResponse
```

## Preconditions

Mock shell scripts in `testdata/`.

## Steps

1. Set `Op=fetch` in leaves.

## Context

Service-layer tests without full daemon HTTP.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch"
	return nil
}
```