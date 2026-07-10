# Scenario

**Feature**: choose Projects empty-area placeholder from loading / count / error

```
# (loading, projectCount, errMsg) -> status row text or ""
loading, count, err -> FormatProjectsListStatusLabel -> Label
```

## Preconditions

`Op=list_status` dispatches to `menubar.FormatProjectsListStatusLabel`.
When `projectCount > 0`, project menus are shown (no status placeholder).

## Steps

1. Set `Op=list_status`.
2. Leaf sets `Loading`, `ProjectCount`, and optional `ErrMsg`.

## Context

REQUIREMENT empty/loading table and scenario 11 (empty+loading is not empty registry).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "list_status"
	return nil
}
```
