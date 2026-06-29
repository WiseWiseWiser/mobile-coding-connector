# Scenario

**Feature**: `git rev-parse` via `/run`

```
remote-agent git -C <repo> rev-parse <ref> -> object id on stdout
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