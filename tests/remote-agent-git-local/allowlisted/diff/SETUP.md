# Scenario

**Feature**: `git diff` via `/run`

```
remote-agent git -C <repo> diff [args] -> patch text on stdout
```

## Preconditions

Leaf prepares modified or staged files.

## Context

Requirement: diff includes `--cached` and path args.

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