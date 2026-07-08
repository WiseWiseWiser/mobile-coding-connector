# Scenario

**Feature**: local-time reset display for dropdown comma layout

```
reset string + now -> FormatResetDisplay -> "{Month} {day}, {HH}:{mm}" | raw passthrough
```

## Preconditions

1. `macosapp/menubar` exports `FormatResetDisplay(reset string, now time.Time) string`.
2. Grok resets parse in Pacific; display converts to `now.Location()`.
3. Codex resets parse in `now.Location()` and reformat to `{Mon} {day}, {HH}:{mm}`.

## Steps

1. Leaf setup sets `Op=reset-display`, `Reset`, and `NowRFC3339` (timezone in offset).

## Context

REQUIREMENT-DESIGN-menubar-display-v2.md `FormatResetDisplay` helper tests.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "reset-display"
	return nil
}
```