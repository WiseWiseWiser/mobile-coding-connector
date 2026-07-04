# Scenario

**Feature**: parse usage lines buried in scrollback noise

```
noisy stdout -> ParseShowUsageOutput -> UsageInfo
```

## Preconditions

`show-usage-noisy.txt` fixture.

## Steps

1. `FixtureFile=show-usage-noisy.txt`.

## Context

REQUIREMENT leaf: `parse/extra-noise`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "show-usage-noisy.txt"
	req.ExpectParseError = false
	return nil
}
```