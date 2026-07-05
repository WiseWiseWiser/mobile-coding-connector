# Scenario

**Feature**: parse error when monthly usage line missing

```
stdout without Monthly usage -> error
```

## Preconditions

`show-status-missing-monthly.txt`.

## Steps

1. `ExpectParseError=true`.

## Context

REQUIREMENT leaf: `parse/missing-monthly`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.FixtureFile = "show-status-missing-monthly.txt"
	req.ExpectParseError = true
	return nil
}
```