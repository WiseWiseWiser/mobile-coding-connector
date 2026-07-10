# Scenario

**Feature**: macOS menu bar Projects submenu formatting and Swift contract

```
ProjectStatus / WorktreeStatus -> FormatProjectTitle / FormatWorktreeTitle -> strings
empty registry -> FormatProjectsEmptyLabel -> "No wrk projects"
Swift sources -> Projects menu + ServerClient /api/wrk/...
```

## Preconditions

1. `macosapp/menubar` exports `FormatProjectTitle`, `FormatWorktreeTitle`, and
   `FormatProjectsEmptyLabel`.
2. Go formatter leaves are pure function calls — no network or subprocess.
3. Client leaves read Swift sources under `macos-ai-critic/ai-critic-macos/`.

## Steps

1. Leaf `Setup` sets `Op` and formatter-specific inputs (or `ClientLeaf` for Swift).
2. Root `Run` dispatches by `Op` to formatters or source inspection.
3. Leaf `Assert` checks exact strings or Swift contract fields.

## Context

Implements REQUIREMENT-DESIGN-wrkserver-projects-menubar.md section B (and
optional C). Primary label logic lives in Go (`macosapp/menubar/`); Swift mirrors
the same contracts. HTTP create/list behavior is covered by `tests/wrkserver/`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	// Root validates request pointer; grouping/leaf Setups populate Op and inputs.
	if req == nil {
		t.Fatal("nil request")
	}
	return nil
}
```
