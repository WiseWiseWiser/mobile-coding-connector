# Scenario

**Feature**: comma-prefixed relative suffix for dropdown reset parentheses

```
reset + now -> FormatResetSuffix -> ", left 3d" | empty
```

## Preconditions

1. `macosapp/menubar` exports `FormatResetSuffix(reset string, now time.Time) string`.
2. Returns `, ` + `FormatTimeLeft` when parseable; empty when not.

## Steps

1. Leaf setup sets `Op=reset-suffix`, `Reset`, and `NowRFC3339`.

## Context

REQUIREMENT-DESIGN-menubar-rel-time.md `dropdown/reset-suffix/` group.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "reset-suffix"
	return nil
}
```