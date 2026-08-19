# Scenario

**Feature**: native-terminals list mode / color flag conflicts and --json batch override

```
--tty + --no-tty -> Error (cannot be specified together)
--color + --no-color -> Error
--json --tty -> still batch Inventory JSON (no split box, no ANSI)
```

## Context

Flag leaves use default CLI Op (RunWithWriters). Seed cache where JSON body is checked.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "cli"
	return nil
}
```
