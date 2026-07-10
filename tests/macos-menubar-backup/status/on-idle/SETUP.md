# Scenario

**Feature**: on + idle status includes last and next relative times

```
enabled, phase=idle, last=now-12m, next=now+48m
  -> "Status: On · last 12m ago · next in 48m"
```

## Preconditions

Task enabled, not running, last finish and next run known.

## Steps

1. Fixed now 15:00Z; last 14:48Z; next 15:48Z.

## Context

REQUIREMENT #12.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = true
	req.Phase = "idle"
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.LastFinishedRFC3339 = "2026-07-10T14:48:00Z" // 12m ago
	req.NextRunRFC3339 = "2026-07-10T15:48:00Z"       // in 48m
	return nil
}
```
