# Scenario

**Feature**: all four command names are equivalent for list -h

```
for name in native-terminals native-terminal native-terms native-term:
  local-agent <name> list -h
    -> exit 0
    -> Usage mentions native-terminals
    -> --json present
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.CommandNames = nativeTerminalCommandNames()
	req.Args = []string{"list", "-h"}
	req.SkipHooks = true
	return nil
}
```
