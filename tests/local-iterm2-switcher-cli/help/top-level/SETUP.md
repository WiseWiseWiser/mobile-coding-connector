# Scenario

**Feature**: native-terminals -h lists list, focus, and aliases

```
local-agent native-terminals -h
  -> Usage local-agent native-terminals
  -> list, focus, Aliases: native-terminal, native-terms, native-term
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "-h"}
	req.SkipHooks = true
	return nil
}
```
