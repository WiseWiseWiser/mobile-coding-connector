# Scenario

**Feature**: parse no-timezone Next reset buried in TUI scrollback noise

```
noisy stdout (no TZ on Next reset) -> ParseShowUsageOutput -> UsageInfo + PT default
```

## Preconditions

`show-usage-noisy-no-tz.txt` fixture.

## Steps

1. `FixtureFile=show-usage-noisy-no-tz.txt`, `ExpectParseError=false`.

## Context

REQUIREMENT leaf: `parse/noisy-no-timezone`.
Classic TDD: RED until no-TZ candidate matches inside noisy corpus.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "show-usage-noisy-no-tz.txt"
	req.ExpectParseError = false
	return nil
}
```
