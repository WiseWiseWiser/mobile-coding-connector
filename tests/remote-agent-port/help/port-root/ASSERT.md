## Expected

1. Exit code 0.
2. Stdout is help for `remote-agent port` and mentions `list` and `visit` subcommands.
3. Trailing newline after last content line.

## Errors

- Exit non-zero.
- Help omits `list` or `visit`.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	if !strings.Contains(strings.ToLower(out), "usage") {
		t.Fatalf("expected Usage in help; stdout:\n%s", out)
	}
	if !strings.Contains(out, "list") || !strings.Contains(out, "visit") {
		t.Fatalf("help must mention list and visit; stdout:\n%s", out)
	}
	if !strings.HasSuffix(out, "\n") && !strings.HasSuffix(out, "\n\n") {
		// help may or may not use Print vs Println; prefer trailing newline
		if len(out) > 0 && !strings.HasSuffix(out, "\n") {
			t.Fatalf("stdout should end with newline; got %q", out[len(out)-20:])
		}
	}
}
```
