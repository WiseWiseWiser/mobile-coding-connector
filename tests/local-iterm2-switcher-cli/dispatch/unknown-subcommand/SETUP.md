# Scenario

**Feature**: unknown subcommand uses canonical native-terminals name

```
local-agent native-term foo
  -> Error: unknown native-terminals subcommand: foo
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-term", "foo"}
	req.SkipHooks = true
	return nil
}
```
