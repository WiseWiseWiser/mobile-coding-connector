# Scenario

**Feature**: bare terminals is removed (unknown command)

```
local-agent terminals list
  -> unknown command: terminals
  -> non-zero
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"terminals", "list"}
	// Hooks only matter while bare `terminals` still exists (fast fail path).
	// After removal, unknown-command returns before inject is used.
	req.ITermDown = true
	return nil
}
```
