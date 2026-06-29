# Scenario

**Feature**: read-only `git remote` via `/run`

```
remote-agent git -C <repo> remote -v -> lists remotes
```

## Preconditions

Harness may configure remotes with local `git` before invoking remote-agent.

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