# Scenario

**Feature**: when projects exist, no empty-area status placeholder

```
FormatProjectsListStatusLabel(loading=true|false, count>0, ...) -> ""
```

## Preconditions

At least one project is available to render as a submenu (even if a refresh is
in flight — rows stay; optional top updating cue is out of this helper).

## Steps

1. Set `ProjectCount=2`, `Loading=true` (refreshing with stale rows), empty error.

## Context

REQUIREMENT: success with rows → project menus only for the empty-area helper
(returns empty string so callers do not replace rows with a status label).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Loading = true
	req.ProjectCount = 2
	req.ErrMsg = ""
	return nil
}
```
