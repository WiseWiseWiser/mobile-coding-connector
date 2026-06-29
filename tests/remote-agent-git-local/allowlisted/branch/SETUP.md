# Scenario

**Feature**: `git branch` list via `/run`

```
remote-agent git -C <repo> branch -> lists branches with current marker
```

## Preconditions

Repo on a named branch.

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