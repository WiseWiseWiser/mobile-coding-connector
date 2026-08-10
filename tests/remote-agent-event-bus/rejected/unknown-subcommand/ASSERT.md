## Expected

1. Non-zero exit.
2. Combined output contains `Error:` about unknown subcommand (or unknown command if event-bus not wired yet — still Error:).

## Errors

- Exit 0.
- Silent failure without Error:.

## Exit Code

non-zero

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
	if resp.ExitCode == 0 {
		t.Fatal("expected non-zero for unknown event-bus subcommand")
	}
	if !strings.Contains(resp.Combined, "Error:") {
		t.Fatalf("expected Error:; combined:\n%s", resp.Combined)
	}
}
```
