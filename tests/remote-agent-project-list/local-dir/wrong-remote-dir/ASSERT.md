## Expected

1. Exit code 0.
2. `Local Dir:        -` when `remote_dir` in config does not equal API `Dir`.

## Side Effects

None.

## Errors

- Binding local path shown for wrong remote_dir.

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
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	if !strings.Contains(out, "Project: local-wrong-dir (local-wdir-001)") {
		t.Fatalf("missing project;\n%s", out)
	}
	assertLocalDirDash(t, out)
}
```