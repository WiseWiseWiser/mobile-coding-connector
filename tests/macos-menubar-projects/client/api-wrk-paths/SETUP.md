# Scenario

**Feature**: ServerClient uses /api/wrk projects and worktrees paths

```
ServerClient -> GET /api/wrk/projects, POST /api/wrk/worktrees
```

## Preconditions

Swift `ServerClient` on port 23712 for business APIs.

## Steps

1. Set `ClientLeaf=api-wrk-paths`.

## Context

REQUIREMENT scenario 18.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ClientLeaf = "api-wrk-paths"
	return nil
}
```
