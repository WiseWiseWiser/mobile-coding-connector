## Expected

1. Exit code 0.
2. Stdout contains `seatalk.message.received`.
3. OpenTTYSession never called (type gate, not payload alone).

## Errors

- Opening on seatalk.message.received.
- Non-zero exit.

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
	if !strings.Contains(resp.Stdout, fixtureTypeSeatalk) {
		t.Fatalf("expected seatalk type printed; stdout:\n%s", resp.Stdout)
	}
	if len(resp.OpenTTYSessionIDs) != 0 {
		t.Fatalf("OpenTTYSession must not run for non-tty events; got %v", resp.OpenTTYSessionIDs)
	}
}
```
