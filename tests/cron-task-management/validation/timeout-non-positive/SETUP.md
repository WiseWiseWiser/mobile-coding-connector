# Scenario

**Feature**: reject create when timeout is ≤ 0 / unlimited

```
# timeout "0" with valid interval schedule
POST -> non-2xx error (timeout must be > 0; no unlimited)
```

## Preconditions

1. Otherwise-valid interval task body with `timeout: "0"`.

## Steps

1. POST RawBody with timeout zero.
2. Assert error; task not listed.

## Context

Validation: timeout always enforced, must be >0.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Action = "create"
	req.RawBody = map[string]any{
		"name":         "bad-timeout",
		"command":      "echo no",
		"scheduleMode": "interval",
		"interval":     "5m",
		"timeout":      "0",
	}
	return nil
}
```
