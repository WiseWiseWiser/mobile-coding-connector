# Scenario

**Feature**: Shared ITermLiveSession decodes live agent fields

```
Shared ITermLiveSession -> agent_runner + grok_session_id
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "client"
	req.ClientLeaf = "live-session-agent-fields"
	return nil
}
```
