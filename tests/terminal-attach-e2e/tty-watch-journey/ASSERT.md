---
label: e2e, slow, heavy
explanation: real server + remote-agent + tty-watch; ~40s including rebuild
---
## Expected

- Harness error is nil (script exit 0).
- `resp.Output` contains `PASS verify-terminal-attach-e2e`.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("e2e gate failed: %v", err)
	}
	if !strings.Contains(resp.Output, "PASS verify-terminal-attach-e2e") {
		t.Fatalf("missing PASS line in:\n%s", resp.Output)
	}
}
```
