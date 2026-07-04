# Scenario

**Feature**: menu-bar grok label formatting (Go spec for Swift client)

```
status + weekly_limit + error -> FormatGrokLabel -> compact label string
```

## Preconditions

1. `macosapp/menubar` exports `FormatGrokLabel(status, weeklyLimit, errorMsg string) string`
   and `TestExported_MaxLabelLen()` for truncation budget.
2. No subprocess or HTTP — pure function calls.

## Steps

1. Leaf `Setup` sets `Status`, `WeeklyLimit`, and `ErrorMsg`.
2. Root `Run` calls `FormatGrokLabel`.
3. Leaf `Assert` checks exact or bounded label text.

## Context

Implements REQUIREMENT-DESIGN-macos-app-and-bar.md Feature 3. Swift UI is manual;
this tree locks the shared formatting contract.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	return nil
}
```