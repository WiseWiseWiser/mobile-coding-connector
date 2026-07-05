# Scenario

**Feature**: unparseable reset string → empty relative text

```
FormatTimeLeft("soon", now) -> ""
```

## Preconditions

Reset string cannot be parsed as grok or codex format.

## Steps

1. Set unparseable reset; `now` is arbitrary fixed value.

## Context

REQUIREMENT leaf: `relative-time/unparseable-fallback`. Dropdown keeps `(Reset soon)` only.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "soon"
	req.NowRFC3339 = "2026-07-06T16:55:00-07:00"
	return nil
}
```