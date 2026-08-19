# Scenario

**Feature**: remote-agent rejects all native-terminals aliases

```
for name in native-terminals native-terminal native-terms native-term:
  remote-agent <name> -> unknown command: <name>
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Profile = "remote"
	req.CommandNames = nativeTerminalCommandNames()
	req.SkipHooks = true
	return nil
}
```
