# Scenario

**Feature**: --tty and --no-tty cannot be combined

```
local-agent native-terminals list --tty --no-tty
  -> Error: --tty and --no-tty cannot be specified together
  -> non-zero exit
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "--tty", "--no-tty"}
	req.SkipHooks = true
	return nil
}
```
