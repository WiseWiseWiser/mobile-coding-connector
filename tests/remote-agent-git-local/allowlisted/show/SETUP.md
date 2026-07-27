# Scenario

**Feature**: `git show` via `/run`

```
remote-agent git -C <repo> show <rev> -> commit metadata and patch
```

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/ai-critic/script/lib"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}
```