## Expected

1. Non-zero exit.
2. Output indicates unauthorized or auth failure.
3. Combined output does **not** contain the start-server hint phrase `Start the server with: ai-critic` (substring `ai-critic` alone may appear in URLs — forbid the hint line pattern).

## Side Effects

Server was reachable; auth rejected bad token.

## Errors

- Start hint shown for auth failure (regression).

## Exit Code

Non-zero.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected auth failure; stdout:\n%s stderr:\n%s", resp.Stdout, resp.Stderr)
	}
	lower := strings.ToLower(resp.Combined)
	if strings.Contains(lower, "start the server with") && strings.Contains(lower, "ai-critic") {
		t.Fatalf("must not show start hint on auth failure; combined:\n%s", resp.Combined)
	}
	if !strings.Contains(lower, "unauthorized") && !strings.Contains(lower, "auth") {
		t.Fatalf("expected auth-related error; combined:\n%s", resp.Combined)
	}
}
```