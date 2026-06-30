# Scenario

**Feature**: Setup page on uninitialized server

```
# no server-credentials; normal (non-quick-test) server mode
Uninitialized server -> Setup page -> Generate Random button
```

## Preconditions

1. Server runs without `--quick-test` and without credentials file.
2. Vite dev server proxies the React frontend.

## Steps

1. Child leaf sets `Request.Uninitialized = true` and supplies `script.js`.

## Context

Grouping for first-launch Setup page UI tests. Quick-test mode bypasses auth and
cannot reproduce the `not_initialized` bug.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 120
	}
	return nil
}
```