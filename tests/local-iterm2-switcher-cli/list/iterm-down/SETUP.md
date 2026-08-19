# Scenario

**Feature**: list warns when iTerm is not running

```
ITermRunning=false (hook)
local-agent native-terminals list -> warning iTerm2 is not running
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Args = []string{"native-terminals", "list"}
	req.ITermDown = true
	return nil
}
```
