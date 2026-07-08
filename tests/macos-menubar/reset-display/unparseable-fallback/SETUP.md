# Scenario

**Feature**: unparseable reset string returned unchanged

```
FormatResetDisplay("soon", now) -> "soon"
```

## Preconditions

Reset string cannot be parsed; formatter must not crash.

## Steps

1. Set unparseable reset and arbitrary fixed `now`.

## Context

REQUIREMENT rule: unparseable reset → return raw string unchanged.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "soon"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```