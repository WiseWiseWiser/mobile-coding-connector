# Scenario

**Feature**: web is local-only

```
agent-run web … -> Error: not available via remote-agent (local-only)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	setCLI(req, "agent-run", "web")
	return nil
}
```
