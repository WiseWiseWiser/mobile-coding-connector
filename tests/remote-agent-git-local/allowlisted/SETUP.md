# Scenario

**Feature**: allowlisted local git subcommands stream successfully

```
# POST /api/remote-agent/git/run -> git subprocess -> stdout/stderr stream
remote-agent git -C <repo> <allowlisted> [args] -> exit mirrors git
```

## Preconditions

Leaf creates a real git repository under a temp directory unless testing errors.

## Steps

1. Leaf builds repo state and calls `setGitLocalArgs` with the subcommand argv tail.
2. Assertions expect exit 0 and git output on stdout.

## Context

Each child narrows on one allowlisted subcommand from the requirement table.

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