## Expected

1. Exit 0.
2. Stdout contains the remote store home path (`resp.StoreHome`) or a clear
   `home:` line that embeds that path.
3. Does not error as unknown subcommand.

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
	if strings.TrimSpace(out) == "" {
		t.Fatalf("bare status should print home path")
	}
	if resp.StoreHome == "" {
		t.Fatalf("harness StoreHome empty")
	}
	if !strings.Contains(out, resp.StoreHome) {
		// Accept home: prefix with path on same or next line after normalization.
		if !strings.Contains(strings.ToLower(out), "home") {
			t.Fatalf("expected home path %q in stdout:\n%s", resp.StoreHome, out)
		}
		t.Fatalf("expected store home %q in bare status stdout:\n%s", resp.StoreHome, out)
	}
}
```
