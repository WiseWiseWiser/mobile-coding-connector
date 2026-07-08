# Scenario

**Feature**: codex reset string reformatted to local display order

```
FormatResetDisplay("08:00 on 1 Aug", now=Jul 6 08:00 PDT) -> "Aug 1, 08:00"
```

## Preconditions

Codex reset is already in `now.Location()`; output uses `{Mon} {day}, {HH}:{mm}`.

## Steps

1. Set codex reset and PDT `now`.

## Context

REQUIREMENT suggested leaf: `08:00 on 1 Aug` → `Aug 1, 08:00`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Reset = "08:00 on 1 Aug"
	req.NowRFC3339 = "2026-07-06T08:00:00-07:00"
	return nil
}
```