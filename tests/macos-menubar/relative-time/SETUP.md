# Scenario

**Feature**: compound relative countdown from provider reset strings

```
reset string + now -> FormatTimeLeft -> left 3d4h | left 4h5m | left 5m | left 0m | empty
```

## Preconditions

1. `macosapp/menubar` exports `FormatTimeLeft(reset string, now time.Time) string`.
2. Tests use fixed `NowRFC3339` with explicit timezone offsets in leaves.

## Steps

1. Leaf setup sets `Op=time-left`, `Reset`, and `NowRFC3339`.

## Context

REQUIREMENT-DESIGN-menubar-display-v2.md compound relative rules: two-tier units
(`d`+`h`, `h`+`m`, `m` only); omit zero tail units; use `m` not `min`;
minutes floor to at least 1 when 0 < duration < 1h; duration ≤ 0 → `left 0m`;
unparseable → empty.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "time-left"
	return nil
}
```