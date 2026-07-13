# Scenario

**Feature**: compose-only dropdown lines from structured API fields

```
# Swift/Go UI concatenates backend tokens — no re-parse of raw next_reset
weekly_limit + reset_display + time_left -> ComposeGrokDropdownLine -> dropdown text
monthly + credits + reset_display + time_left -> ComposeCodexDropdownLine -> dropdown text
```

## Preconditions

1. `macosapp/menubar` exports `ComposeGrokDropdownLine` and
   `ComposeCodexDropdownLine` (implementer; classic TDD RED until present).
2. Inputs are already-structured API fields (`reset_display`, `time_left`), not
   raw provider `next_reset` strings.
3. No network, no clock, no `FormatResetDisplay` / `FormatTimeLeft` in this path.

## Steps

1. Leaf `Setup` sets `Op` and structured field strings.
2. Root `Run` dispatches to compose helpers.
3. Leaf `Assert` checks exact dropdown line text.

## Context

REQUIREMENT-DESIGN-usage-structured-reset-ab.md scenarios 5–6. Existing
`dropdown/*` leaves remain low-level tests of raw-parse producers
(`FormatGrokDropdownLine` / `FormatResetDisplay` / `FormatTimeLeft`).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Status == "" {
		req.Status = "ready"
	}
	return nil
}
```
