# Scenario

**Feature**: argv includes non-default port and token flags

```
# keep-alive forward
cfg{Port:30001,Token:tok,Disabled:false}
  -> ... --event-bus-publish-port 30001 --event-bus-publish-token tok
```

## Steps

1. PortFlag 30001, TokenFlag `tok-abc`, NoPublish false.
2. BaseArgs seed preserved; flags appended.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	req.BaseArgs = []string{"ai-critic-server", "--port", "8080"}
	req.PortFlag = 30001
	req.TokenFlag = "tok-abc"
	req.NoPublish = false
	return nil
}
```
