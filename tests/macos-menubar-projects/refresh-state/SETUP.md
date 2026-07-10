# Scenario

**Feature**: pure Projects list state transitions (stale-while-revalidate)

```
# start / success / failure reducers — never clear projects on start or fail
ProjectsListState + action -> ApplyProjectsRefresh* -> ProjectsListState
```

## Preconditions

`Op=refresh_state` applies `menubar.ApplyProjectsRefreshStart|Success|Failure`
to a `ProjectsListState` built from `PriorProjects`, `PriorLoading`, `PriorError`.

Project tokens are basename strings for test simplicity (not full status structs).

## Steps

1. Set `Op=refresh_state`.
2. Leaf sets prior state, `RefreshAction`, and success/failure payloads.

## Context

REQUIREMENT scenarios 8–10 (refresh start / fail / success).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "refresh_state"
	return nil
}
```
