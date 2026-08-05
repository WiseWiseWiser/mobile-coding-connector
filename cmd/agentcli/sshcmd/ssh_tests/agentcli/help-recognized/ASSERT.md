## Expected

1. Harness `err` is nil.
2. `AgentcliErr` is empty (help returns nil).
3. `UnknownCommand` is false.

## Side Effects

- Help text may print to process stdout (not asserted here).

## Errors

None expected.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.UnknownCommand {
		t.Fatalf("ssh must not be unknown command; AgentcliErr=%q", resp.AgentcliErr)
	}
	if resp.AgentcliErr != "" {
		t.Fatalf("ssh --help should return nil error; got %q", resp.AgentcliErr)
	}
}
```
