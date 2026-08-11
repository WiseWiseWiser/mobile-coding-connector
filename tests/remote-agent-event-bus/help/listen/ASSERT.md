## Expected

1. Exit code 0.
2. Stdout documents listen and flags among: `--type`, `--json`, `--replay`,
   `--open-tty` (default off; open iTerm attach on agent.tty.started).
3. Stdout contains `Usage`.

## Errors

- Missing flag documentation (including `--open-tty`).
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
	for _, needle := range []string{"listen", "--type", "--json", "--replay", "--open-tty"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("listen help missing %q; stdout:\n%s", needle, out)
		}
	}
}
```
