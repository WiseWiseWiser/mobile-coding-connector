# Scenario

**Feature**: --tty paints split from cache then q quits without focus

```
SeedCache (sess-a)
SetTerminalsKeys("q")
native-terminals list --tty
  -> exit 0, stdout has split + grok review + cached|up to date|incremental
  -> FocusCalled=false
  -> CaptureCalls=0 when layout same
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "--tty"}
	req.SeedCache = true
	req.Keys = "q"
	return nil
}
```
