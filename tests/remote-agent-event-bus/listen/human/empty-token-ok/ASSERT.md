## Expected

1. Exit code 0.
2. Connected + event type present (token empty is accepted).
3. No auth Error forced by empty token alone.

## Errors

- Fail solely because Token is empty.
- Missing event output.

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
		t.Fatalf("empty token should still listen; exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "connected") {
		t.Fatalf("expected connected; stdout:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, fixtureTypeSeatalk) {
		t.Fatalf("expected event type; stdout:\n%s", resp.Stdout)
	}
}
```
