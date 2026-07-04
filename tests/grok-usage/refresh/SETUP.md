# Scenario

**Feature**: grok usage refresh skips overlapping fetches

```
slow fake TUI + concurrent TriggerRefresh -> single PTY session
```

## Preconditions

`mock-slow.sh` fake TUI sleeps 2s after `/usage show` and increments `GROK_MOCK_COUNTER_FILE`.

## Steps

1. Set `Op=refresh` in leaf.

## Context

Validates skip-concurrent-fetch requirement.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "refresh"
	return nil
}
```