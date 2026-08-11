## Expected

1. Exit code 0.
2. Event printed (`agent.tty.started`).
3. OpenTTYSession never called (flag default off).

## Errors

- Any open call when OpenTTY is false.
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
	if !strings.Contains(resp.Stdout, fixtureTypeTTY) {
		t.Fatalf("expected log-only tty event; stdout:\n%s", resp.Stdout)
	}
	if len(resp.OpenTTYSessionIDs) != 0 {
		t.Fatalf("without OpenTTY, OpenTTYSession must never run; got %v", resp.OpenTTYSessionIDs)
	}
}
```
