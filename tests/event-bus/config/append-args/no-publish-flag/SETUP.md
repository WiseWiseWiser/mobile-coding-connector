# Scenario

**Feature**: argv includes --no-event-bus-publish when disabled

```
# disabled config
cfg{Disabled:true} -> ... --no-event-bus-publish
```

## Steps

1. NoPublish true.
2. Assert flag present on ArgsOut.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.BaseArgs = []string{"ai-critic-server"}
	req.PortFlag = 0
	req.TokenFlag = ""
	req.NoPublish = true
	return nil
}
```
