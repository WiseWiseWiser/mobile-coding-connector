# Scenario

**Feature**: local-agent native-terminals list is in-process batch (file last-good)

```
# seed optional complete last-good under HOME
local-agent native-terminals list [--json] [--refresh]
  -> Capture/Layout hooks + cache file (no HTTP)
  -> print human or JSON once
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	if len(req.Args) == 0 {
		req.Args = []string{"native-terminals", "list"}
	}
	return nil
}
```
