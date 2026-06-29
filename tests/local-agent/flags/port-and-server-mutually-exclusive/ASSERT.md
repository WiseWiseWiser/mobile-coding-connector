## Expected

1. Non-zero exit code.
2. Combined output mentions both `--port` and `--server` (or states they cannot be used together).
3. Output does not report a successful ping.

## Side Effects

No server subprocess; no successful API call.

## Errors

- Process exits 0.
- Error text is empty or unrelated to flag conflict.

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
		t.Fatalf("expected non-zero exit for --port + --server; combined:\n%s", resp.Combined)
	}
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "port") || !strings.Contains(lower, "server") {
		t.Fatalf("expected usage error mentioning port and server; got:\n%s", resp.Combined)
	}
	if strings.Contains(resp.Stdout, "pong") || strings.Contains(resp.Stdout, "Status: ok") {
		t.Fatalf("ping should not succeed when flags conflict; stdout:\n%s", resp.Stdout)
	}
}
```