# Scenario

**Feature**: create rejects missing project_path

```
# body without project_path
CreateWorktree({}) -> 4xx {"error":"..."}
```

## Preconditions

No `ProjectPath`; omit task.

## Steps

1. Leave `ProjectPath` empty and `OmitTask=true` so body is `{}`.
2. POST create.

## Context

REQUIREMENT scenario 8.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ProjectPath = ""
	req.OmitTask = true
	return nil
}
```
