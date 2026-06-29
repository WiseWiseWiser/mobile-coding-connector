# Scenario

**Feature**: `git log` via `/run`

```
remote-agent git -C <repo> log <flags> -> commit history on stdout
```

## Preconditions

Leaf creates commit history.

## Steps

Descendant leaves call `setGitLocalArgs` with `log` and flags.

## Context

Requirement: `--oneline`, `-n`, etc.

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