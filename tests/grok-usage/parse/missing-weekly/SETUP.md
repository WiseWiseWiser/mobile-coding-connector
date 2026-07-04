# Scenario

**Feature**: parse error when weekly limit line missing

```
stdout without Weekly limit -> error
```

## Preconditions

`show-usage-missing-weekly.txt`.

## Steps

1. `ExpectParseError=true`.

## Context

REQUIREMENT leaf: `parse/missing-weekly`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "show-usage-missing-weekly.txt"
	req.ExpectParseError = true
	return nil
}
```