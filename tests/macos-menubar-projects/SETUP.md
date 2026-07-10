# Scenario

**Feature**: macOS Projects menu title parts, loading labels, and stale-while-revalidate list state

```
# title parts: basename left, decoration right
ProjectStatus -> FormatProjectTitleParts -> {Leading, Trailing}
WorktreeStatus -> FormatWorktreeTitleParts -> {Leading, Trailing}
legacy Title = Leading + "  " + Trailing

# empty-area labels
() -> FormatProjectsLoadingLabel / Empty / LoadFailed -> placeholder text
(loading, count, err) -> FormatProjectsListStatusLabel -> which placeholder

# list state (pure)
ProjectsListState -> ApplyProjectsRefreshStart|Success|Failure -> new state

# Swift contracts
AICriticApp / ProjectsMenuFormatter -> HStack titles, projectsLoading, Loading…
```

## Preconditions

1. `macosapp/menubar` exports title-parts formatters, loading/empty/failed labels,
   list-status selector, and `ProjectsListState` refresh helpers (under test;
   absent symbols keep the tree RED until implementation).
2. Go leaves are pure function / pure reducer calls — no network or subprocess.
3. Client leaves read Swift sources under `macos-ai-critic/`.

## Steps

1. Leaf `Setup` sets `Op` and scenario-specific inputs (or `ClientLeaf` for Swift).
2. Root `Run` dispatches by `Op` to formatters, list status, reducer, or source inspection.
3. Leaf `Assert` checks Leading/Trailing, labels, state snapshot, or Swift contract fields.

## Context

Implements REQUIREMENT-DESIGN-projects-menu-loading-align.md. Extends the prior
Projects menubar formatter tree (single-string titles + empty label) with left/right
parts, loading/failure UX, and stale-while-revalidate. HTTP create/list remains
covered by `tests/wrkserver/`.

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
