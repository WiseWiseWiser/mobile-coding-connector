# Scenario

**Feature**: native-terminals help is local-only discoverability (no daemon, no HTTP wording)

```
local-agent native-terminals [-h] | list -h | focus -h
  -> usage on stdout; no /api, GET, POST, server, stream URL
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.SkipHooks = true
	return nil
}
```
