# Scenario

**Feature**: parse Grok Next reset without timezone (local wall clock)

```
Weekly limit + Next reset (no TZ) -> UsageInfo with PT default
```

## Preconditions

`show-usage-no-timezone.txt` fixture (current Grok 0.2.99 shape).

## Steps

1. `FixtureFile=show-usage-no-timezone.txt`, `ExpectParseError=false`.

## Context

REQUIREMENT leaf: `parse/no-timezone`.
No-TZ form stays bare (`July 17, 08:55`); consumers treat as local time.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "show-usage-no-timezone.txt"
	req.ExpectParseError = false
	return nil
}
```
