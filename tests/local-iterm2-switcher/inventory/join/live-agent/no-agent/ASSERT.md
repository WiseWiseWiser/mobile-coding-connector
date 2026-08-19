## Expected

1. UUID rematch still stars the live row.
2. Live JSON agent fields stay empty — they come from the snap, not the item.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.FirstBookmarked {
		t.Fatal("live row should rematch by iterm_session_id")
	}
	if resp.FirstAgentRunner != "" || resp.FirstGrokSessionID != "" {
		t.Fatalf("agent_runner=%q grok_session_id=%q want empty", resp.FirstAgentRunner, resp.FirstGrokSessionID)
	}
}
```
