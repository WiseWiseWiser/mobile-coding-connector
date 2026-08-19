# Scenario

**Feature**: --json always batch even with --tty (no split box, no ANSI)

```
SeedCache (sess-a)
local-agent native-terminals list --json --tty
  -> stdout is Inventory JSON with sess-a
  -> no box-drawing ┌, no ANSI
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "--json", "--tty"}
	req.SeedCache = true
	return nil
}
```
