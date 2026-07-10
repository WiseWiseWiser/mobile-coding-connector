# Scenario

**Feature**: idle empty registry shows No wrk projects

```
FormatProjectsListStatusLabel(loading=false, count=0, err="") -> "No wrk projects"
```

## Preconditions

Load finished successfully with zero projects.

## Steps

1. Set `Loading=false`, `ProjectCount=0`, empty `ErrMsg`.

## Context

REQUIREMENT: `!loading && empty && error==nil` → `No wrk projects`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Loading = false
	req.ProjectCount = 0
	req.ErrMsg = ""
	return nil
}
```
