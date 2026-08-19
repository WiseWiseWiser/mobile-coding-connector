# Scenario

**Feature**: --color and --no-color cannot be combined

```
local-agent native-terminals list --color --no-color
  -> Error: --color and --no-color cannot be specified together
  -> non-zero exit
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list", "--color", "--no-color"}
	req.SkipHooks = true
	return nil
}
```
