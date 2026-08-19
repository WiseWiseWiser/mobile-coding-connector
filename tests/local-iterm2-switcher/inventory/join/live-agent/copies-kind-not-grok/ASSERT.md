## Expected

1. Live `agent_runner` is the snap Agent kind (`codex`).
2. `grok_session_id` is empty when kind is not grok.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FirstAgentRunner != "codex" {
		t.Fatalf("agent_runner=%q want codex", resp.FirstAgentRunner)
	}
	if resp.FirstGrokSessionID != "" {
		t.Fatalf("grok_session_id=%q want empty", resp.FirstGrokSessionID)
	}
}
```
