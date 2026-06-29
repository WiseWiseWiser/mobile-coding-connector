## Expected

1. Exit 0.
2. Stdout contains `origin` and `https://example.com/foo.git` for fetch and push.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; stdout:\n%s stderr:\n%s", resp.ExitCode, resp.Stdout, resp.Stderr)
	}
	out := resp.Stdout
	for _, want := range []string{"origin", "https://example.com/foo.git"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q;\nstdout=%q\nstderr=%q", want, out, resp.Stderr)
		}
	}
}
```