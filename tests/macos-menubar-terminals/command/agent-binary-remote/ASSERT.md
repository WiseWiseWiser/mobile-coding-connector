## Expected

1. `AgentBinary` is exactly `remote-agent`.

## Errors

- Returning `local-agent` for the remote app profile.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AgentBinary != "remote-agent" {
		t.Fatalf("agent binary = %q, want %q", resp.AgentBinary, "remote-agent")
	}
}
```
