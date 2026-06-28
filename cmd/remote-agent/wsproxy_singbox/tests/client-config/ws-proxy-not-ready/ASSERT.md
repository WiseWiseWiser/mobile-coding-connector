## Expected

- `RunErr` is non-nil and mentions ws-proxy not running.
- `FetchVMessCalled` is true.
- No valid config JSON on stdout.

## Side Effects

- No sing-box config written.

## Errors

- CLI surfaces not-running message (may include start hint).

## Exit Code

- Failure.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr == nil {
		t.Fatal("expected client-config to fail when ws-proxy not ready")
	}
	if !resp.FetchVMessCalled {
		t.Fatal("FetchVMess must be attempted")
	}
	msg := strings.ToLower(resp.RunErr.Error())
	if !strings.Contains(msg, "not running") {
		t.Fatalf("error = %q, want not-running message", resp.RunErr)
	}
	if strings.TrimSpace(resp.Stdout) != "" {
		t.Fatalf("stdout should be empty on error; got %q", resp.Stdout)
	}
}
```