# Scenario

**Feature**: unparseable reset → empty suffix (no comma)

```
FormatResetSuffix("soon", now) -> ""
```

## Preconditions

Unparseable reset must not add `, left …` to dropdown parentheses.

## Steps

1. Set unparseable reset and fixed `now`.

## Context

REQUIREMENT leaf: `dropdown/reset-suffix/unparseable`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "soon"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```