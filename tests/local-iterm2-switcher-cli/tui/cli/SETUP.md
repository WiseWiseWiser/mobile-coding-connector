# Scenario

**Feature**: CLI --tty forces TUI with scripted keys (testhooks input)

```
SeedCache -> native-terminals list --tty
SetTerminalsKeys("q"|"\\r")
  -> paint split + cached; q exits 0 no Focus
  -> Enter focuses sess-a
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	req.SeedCache = true
	return nil
}
```
