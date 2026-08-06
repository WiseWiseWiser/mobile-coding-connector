## Expected

1. `RunErr` empty.
2. Stdout contains `remote-agent sync` and `unison`.
3. Trailing newline.

## Side Effects

- None required.

## Errors

- None.

## Exit Code

- Nil error.

```go
import (
	"strings"
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
	if resp.RunErr != "" {
		t.Fatalf("RunCLI error: %s", resp.RunErr)
	}
	if !strings.Contains(resp.Stdout, "remote-agent sync") {
		t.Fatalf("stdout missing sync usage; got:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "unison") {
		t.Fatalf("stdout missing unison; got:\n%s", resp.Stdout)
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("stdout must end with trailing newline; got %q", resp.Stdout)
	}
}
```
