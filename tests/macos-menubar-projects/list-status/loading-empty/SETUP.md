# Scenario

**Feature**: empty registry while loading shows Loading… not empty label

```
FormatProjectsListStatusLabel(loading=true, count=0, err="") -> "Loading…"
```

## Preconditions

First load or refresh with no cached projects; request still in flight.

## Steps

1. Set `Loading=true`, `ProjectCount=0`, empty `ErrMsg`.

## Context

REQUIREMENT: `projectsLoading && projects.isEmpty` → `Loading…` (not empty registry).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Loading = true
	req.ProjectCount = 0
	req.ErrMsg = ""
	return nil
}
```
