# Scenario

**Feature**: focus without session-id is a usage error

```
local-agent native-terminals focus
  -> Error: (session-id required)
  -> FocusCalled=false
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "focus"}
	return nil
}
```
