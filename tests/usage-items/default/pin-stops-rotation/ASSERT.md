## Expected

1. The response reports `cc-v2` with rotation off.
2. The registry file records the same choice, so the app picks it up on its next poll.

## Errors

- Leaving rotation on, or only holding the choice in memory.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Default != "cc-v2" || resp.Rotate {
		t.Fatalf("selection = %q/rotate=%v", resp.Default, resp.Rotate)
	}
	reg := readRegistry(t, resp.Registry)
	if reg.Default != "cc-v2" || reg.Rotate {
		t.Fatalf("persisted selection = %q/rotate=%v", reg.Default, reg.Rotate)
	}
}
```
