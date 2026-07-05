# Scenario

**Feature**: relative countdown from provider reset strings

```
reset string + now -> FormatTimeLeft -> left 3d | left 3h | left 2min | left 0min | empty
```

## Preconditions

1. `macosapp/menubar` exports `FormatTimeLeft(reset string, now time.Time) string`.
2. Tests use fixed `NowRFC3339` in `America/Los_Angeles` (PDT in July 2026).

## Steps

1. Leaf setup sets `Op=time-left`, `Reset`, and `NowRFC3339`.

## Context

REQUIREMENT-DESIGN-menubar-rel-time.md relative-time rules: floor to largest unit;
minutes floor to at least 1 when 0 < duration < 1h; duration ≤ 0 → `left 0min`;
unparseable → empty.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "time-left"
	return nil
}
```