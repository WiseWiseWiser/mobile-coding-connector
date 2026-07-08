## Expected

1. `Response.RestartLoopSeen` is true OR merged logs contain `failed to become ready`.
2. With 15s core delay and 10s timeout, `Response.ServerReady` may remain false
   (port not open within timeout) — either outcome documents the negative control.

## Side Effects

- Daemon may kill and respawn the managed server during observation.

## Errors

- Build or daemon start failure fails the test immediately.

## Exit Code

- `0` when startup-timeout failure is observed (negative control passes).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	hasFailure := resp.RestartLoopSeen ||
		strings.Contains(resp.DaemonLogs, "failed to become ready")
	if !hasFailure {
		t.Fatal("expected startup-timeout failure with 15s core delay and 10s timeout; got stable ready")
	}
}
```