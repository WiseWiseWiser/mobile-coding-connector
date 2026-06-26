## Expected

1. `HasResultLine` is true with unhealthy result (`Result: unhealthy`).
2. `ExitCode` is non-zero (typically 1).
3. Stdout contains at least one `[fail]` check line.

## Side Effects

None beyond harness teardown.

## Exit Code

Must be non-zero.

## Errors

- Exit code 0 despite unhealthy result.
- Missing `[fail]` lines when unhealthy.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !strings.Contains(resp.Stdout, "Result: unhealthy") {
		t.Fatalf("expected unhealthy result; stdout:\n%s", resp.Stdout)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("ExitCode = 0, want non-zero for unhealthy doctor")
	}
	if !strings.Contains(resp.Stdout, "[fail]") {
		t.Fatalf("expected [fail] check line in unhealthy output; stdout:\n%s", resp.Stdout)
	}
}
```
