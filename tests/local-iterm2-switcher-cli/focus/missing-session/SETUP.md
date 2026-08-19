# Scenario

**Feature**: focus unknown session-id is an error

```
local-agent native-terminals focus no-such-session
  -> Error: on stderr, non-zero
  -> Focus must not succeed (no "focused")
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "focus", "no-such-session"}
	req.SeedCache = true
	return nil
}
```
