# Scenario

**Feature**: list --refresh forces a full recapture (even when file warm)

```
SeedCache (sess-a)
Capture fixture includes sess-b when SecondSnapB
local-agent native-terminals list --json --refresh
  -> CaptureCalls ≥ 1
  -> JSON includes sess-b
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "--json", "--refresh"}
	req.SeedCache = true
	req.SecondSnapB = true
	return nil
}
```
