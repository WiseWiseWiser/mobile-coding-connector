# Scenario

**Feature**: --tty + Enter focuses selected session and exits 0

```
SeedCache (sess-a)
SetTerminalsKeys("\\r")  // Enter
native-terminals list --tty
  -> exit 0, FocusCalled, FocusSession=sess-a
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "--tty"}
	req.SeedCache = true
	req.Keys = "\r"
	return nil
}
```
