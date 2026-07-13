# Scenario

**Feature**: parse Next reset with explicit UTC timezone

```
Weekly limit + Next reset … UTC -> UsageInfo preserves UTC
```

## Preconditions

`show-usage-utc.txt` fixture.

## Steps

1. `FixtureFile=show-usage-utc.txt`, `ExpectParseError=false`.

## Context

REQUIREMENT leaf: `parse/explicit-utc`.
Multi-format priority: explicit UTC (stricter) before no-TZ default.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "show-usage-utc.txt"
	req.ExpectParseError = false
	return nil
}
```
