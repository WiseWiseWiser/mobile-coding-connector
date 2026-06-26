# Scenario

**Feature**: early checks print before delayed upstream fetch

```
# server hook delays upstream_fetch 200ms
configuration load line at t0; upstream proxy fetch line at t0+200ms
```

## Preconditions

`SetTestUpstreamFetchDelay(200ms)` on server via shared test hook.

## Steps

1. Set `UpstreamFetchDelayMs = 200`.
2. Enable `RecordLineTimes`.

## Context

Requirement scenario 8 — `doctor-integration-prints-incrementally`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UpstreamFetchDelayMs = 200
	req.RecordLineTimes = true
	return nil
}
```
