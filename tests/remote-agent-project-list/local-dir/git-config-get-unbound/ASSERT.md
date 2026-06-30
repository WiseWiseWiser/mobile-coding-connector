## Expected

1. Exit code 0.
2. `Local Dir:        -` in git-config get output.

## Side Effects

None.

## Errors

- Spurious bound path.

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
	if !strings.Contains(out, "Project: local-git-config-unbound (local-gcfg-u-001)") {
		t.Fatalf("missing project;\n%s", out)
	}
	assertLocalDirDash(t, out)
}
```