## Expected

1. Exit code 0.
2. Stdout documents listen and flags among: `--type`, `--json`, `--replay`.
3. Stdout contains `Usage`.

## Errors

- Missing flag documentation.
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
	out := resp.Stdout
	if !strings.Contains(strings.ToLower(out), "usage") {
		t.Fatalf("expected Usage in listen help; stdout:\n%s", out)
	}
	for _, needle := range []string{"listen", "--type", "--json", "--replay"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("listen help missing %q; stdout:\n%s", needle, out)
		}
	}
}
```
