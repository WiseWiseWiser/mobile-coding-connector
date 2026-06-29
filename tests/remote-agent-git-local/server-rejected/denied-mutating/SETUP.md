# Scenario

**Feature**: mutating git subcommands denied by allowlist

```
remote-agent git -C <repo> <mutating> -> rejected before git spawn
```

## Preconditions

Valid git repository for every child leaf.

## Steps

1. Leaf creates repo state appropriate to the mutating command.
2. Invokes denied argv via `setGitLocalArgs`.

## Context

Requirement: `add`, `commit`, `checkout`, … out of scope; server/CLI must block.

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

func mkDeniedRepo(t *testing.T) string {
	t.Helper()
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	return dir
}
```