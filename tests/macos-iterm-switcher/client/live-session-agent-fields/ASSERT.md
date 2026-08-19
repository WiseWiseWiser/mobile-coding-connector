## Expected

1. Shared `ITermLiveSession` decodes `agent_runner` and `grok_session_id`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasLiveSessionAgentFields {
		t.Fatal("ITermLiveSession must decode agent_runner and grok_session_id")
	}
}
```
