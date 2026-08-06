## Expected

1. `PreErr`/`RunErr` empty.
2. Stdout contains pair name `mad-max`, local path, and remote path.
3. Prefer at least one of: `prefer`, `newer`, `localHostname`, or `remote-agent-mad-max`.

## Side Effects

- None beyond pre-init.

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
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.PreErr != "" {
		t.Fatalf("pre-init failed: %s", resp.PreErr)
	}
	if resp.RunErr != "" {
		t.Fatalf("show error: %s", resp.RunErr)
	}
	out := resp.Stdout
	for _, needle := range []string{"mad-max", req.LocalPath, req.RemotePath} {
		if !strings.Contains(out, needle) {
			t.Fatalf("show stdout missing %q; got:\\n%s", needle, out)
		}
	}
}
```
