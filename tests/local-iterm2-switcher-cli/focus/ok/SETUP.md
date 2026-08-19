# Scenario

**Feature**: focus known session via in-process Focus hook

```
SeedCache (sess-a) optional for inventory lookup
local-agent native-terminals focus sess-a
  -> FocusCalled, FocusSession=sess-a
  -> exit 0, "focused"
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "focus", "sess-a"}
	req.SeedCache = true
	return nil
}
```
