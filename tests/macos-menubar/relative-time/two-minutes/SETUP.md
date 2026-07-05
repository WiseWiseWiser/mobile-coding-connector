# Scenario

**Feature**: grok reset <1h away → minutes unit

```
FormatTimeLeft("July 6, 16:57 PT", now=Jul 6 16:55 PDT) -> "left 2min"
```

## Preconditions

Remaining duration is 2 minutes.

## Steps

1. Set reset and fixed `now`.

## Context

REQUIREMENT leaf: `relative-time/two-minutes`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "July 6, 16:57 PT"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```