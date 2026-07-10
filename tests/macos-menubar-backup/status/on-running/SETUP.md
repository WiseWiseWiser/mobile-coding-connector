# Scenario

**Feature**: on + running status title

```
enabled, phase=running -> "Status: On · Running"
```

## Preconditions

A backup is in progress for the active server.

## Steps

1. Enabled=true, Phase=running.

## Context

REQUIREMENT #13.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = true
	req.Phase = "running"
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	return nil
}
```
