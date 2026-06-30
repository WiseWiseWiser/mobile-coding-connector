## Expected

1. Exit code 0.
2. Stdout contains `Local Dir:        -` for the project block.

## Side Effects

None.

## Errors

- Bound path shown without binding.

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
	if !strings.Contains(out, "Project: local-unbound (local-unbound-001)") {
		t.Fatalf("missing project header;\n%s", out)
	}
	assertLocalDirDash(t, out)
}
```