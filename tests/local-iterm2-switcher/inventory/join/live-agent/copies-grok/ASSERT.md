## Expected

1. First live session JSON has `agent_runner=grok`.
2. First live session JSON has `grok_session_id=g1`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FirstAgentRunner != "grok" {
		t.Fatalf("agent_runner=%q want grok", resp.FirstAgentRunner)
	}
	if resp.FirstGrokSessionID != "g1" {
		t.Fatalf("grok_session_id=%q want g1", resp.FirstGrokSessionID)
	}
}
```
