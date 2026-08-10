## Expected

1. Exit code 0.
2. Stdout contains `seatalk.message.received`.
3. Stdout does **not** contain `agent.tty.started`.
4. Connected line may still appear.

## Errors

- Non-matching type printed.
- Matching type missing.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	if !strings.Contains(out, fixtureTypeSeatalk) {
		t.Fatalf("expected matching type %q; stdout:\n%s", fixtureTypeSeatalk, out)
	}
	if strings.Contains(out, fixtureTypeTTY) {
		t.Fatalf("non-matching type %q must be filtered out; stdout:\n%s", fixtureTypeTTY, out)
	}
}
```
