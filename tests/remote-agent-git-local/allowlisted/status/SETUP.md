# Scenario

**Feature**: `git status` via `/run`

```
remote-agent git -C <repo> status -> human-readable status on stdout
```

## Preconditions

Leaf supplies worktree state.

## Steps

Descendant leaves call `setGitLocalArgs(..., "status")`.

## Context

Primary reported bug in requirement.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```