# Scenario

**Feature**: watch missing session errors

```
WatchInject=error -> agent-run watch sess-nope -> error
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "watch", "sess-nope")
	req.Seeds = nil
	req.WatchInject = "error"
	return nil
}
```
