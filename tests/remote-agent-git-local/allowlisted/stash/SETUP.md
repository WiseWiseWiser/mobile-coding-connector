# Scenario

**Feature**: `git stash list` via `/run`

```
remote-agent git -C <repo> stash list -> stash entries or empty
```

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