## Expected

1. `RunErr` empty.
2. Stdout contains `doctor` and `status`.
3. Trailing newline.

## Side Effects

- None.

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
	out := resp.Stdout
	for _, needle := range []string{"doctor", "status"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("stdout missing %q; got:\n%s", needle, out)
		}
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("stdout must end with trailing newline; got %q", out)
	}
}
```
