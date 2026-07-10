# Scenario

**Feature**: on + error status title with relative last failure time

```
enabled, phase=error, lastFinished=now-5m -> "Status: On · Error · 5m ago"
```

## Preconditions

Last run failed; last_error / last_finished set. Preferred presentation when last_error set.

## Steps

1. Enabled=true, Phase=error, last finish 5m ago, LastError non-empty.

## Context

REQUIREMENT status title (optional preferred error form).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Enabled = true
	req.Phase = "error"
	req.LastError = "download failed"
	req.NowRFC3339 = "2026-07-10T15:00:00Z"
	req.LastFinishedRFC3339 = "2026-07-10T14:55:00Z" // 5m ago
	return nil
}
```
