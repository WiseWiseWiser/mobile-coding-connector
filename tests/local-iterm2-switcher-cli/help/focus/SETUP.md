# Scenario

**Feature**: native-terminals focus -h documents session-id

```
local-agent native-terminals focus -h -> session-id (no POST)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "focus", "-h"}
	req.SkipHooks = true
	return nil
}
```
