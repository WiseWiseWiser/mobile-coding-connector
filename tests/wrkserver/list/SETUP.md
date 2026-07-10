# Scenario

**Feature**: GET list projects handler

```
# list recorded mains + linked worktree status
ListProjects -> 200 {"projects":[...]}
```

## Preconditions

`Op=list` dispatches to `Server.ListProjects` via httptest.

## Steps

1. Set `Op` to `list`.
2. Leaf seeds registry and git state under `WrkHome`.

## Context

REQUIREMENT scenarios 1–4.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "list"
	return nil
}
```
