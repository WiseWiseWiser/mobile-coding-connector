# Scenario

**Feature**: native-terminals list -h documents Usage, --json, --refresh

```
local-agent native-terminals list -h
  -> Usage local-agent native-terminals list
  -> --json --refresh (no GET/stream URL)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "-h"}
	req.SkipHooks = true
	return nil
}
```
