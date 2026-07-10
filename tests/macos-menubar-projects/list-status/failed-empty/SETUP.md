# Scenario

**Feature**: empty list after load failure shows Failed to load projects

```
FormatProjectsListStatusLabel(loading=false, count=0, err="timeout") -> "Failed to load projects"
```

## Preconditions

Load finished with error; no prior projects to keep showing.

## Steps

1. Set `Loading=false`, `ProjectCount=0`, non-empty `ErrMsg`.

## Context

REQUIREMENT: `!loading && empty && error!=nil` → `Failed to load projects`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Loading = false
	req.ProjectCount = 0
	req.ErrMsg = "timeout"
	return nil
}
```
