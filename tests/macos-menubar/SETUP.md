# Scenario

**Feature**: menu-bar usage label and dropdown formatting (Go spec for Swift client)

```
usage fields -> FormatGrokLabel / FormatMenuBarLabel / dropdown formatters -> strings
```

## Preconditions

1. `macosapp/menubar` exports `FormatGrokLabel`, `FormatMenuBarLabel`,
   `FormatGrokDropdownLine`, `FormatCodexDropdownLine`, `FormatTimeLeft`,
   `FormatResetDisplay`, and `TestExported_MaxLabelLen()`.
2. No subprocess or HTTP — pure function calls.

## Steps

1. Leaf `Setup` sets `Op` and formatter-specific inputs.
2. Root `Run` dispatches by `Op` (default `grok-label` for legacy leaves).
3. Leaf `Assert` checks exact label or dropdown line text.

## Context

Implements REQUIREMENT-DESIGN-macos-app-and-bar.md Feature 3,
REQUIREMENT-DESIGN-codex-usage.md Part 2 menubar formatters, and
REQUIREMENT-DESIGN-menubar-display-v2.md dropdown v2 (local reset display,
compound relative countdown). Swift UI is manual; this tree locks the shared
formatting contract.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```