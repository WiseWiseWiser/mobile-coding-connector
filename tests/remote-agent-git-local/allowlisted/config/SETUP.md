# Scenario

**Feature**: read-only `git config` via `/run`

```
remote-agent git -C <repo> config --get <key> -> value on stdout
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