# Scenario

**Feature**: warm list uses file last-good + Layout only (no deep Capture)

```
SeedCache (sess-a) under HOME
local-agent native-terminals list
  -> Layout increment (CaptureCalls=0)
  -> human: Desktop 1, grok review, incremental | up to date
  -> no localhost /api leak
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list"}
	req.SeedCache = true
	return nil
}
```
