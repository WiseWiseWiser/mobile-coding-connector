## Expected

1. `Run` completes without error.
2. `Response.DaemonStatus` is non-nil.
3. `DaemonStatus.KeepAlivePID` is greater than zero.
4. `DaemonStatus.KeepAlivePort` equals `23312`.
5. `DaemonStatus.ServerPort` equals `Response.ServerPort`.
6. `DaemonStatus.Running` is true (managed server PID present).

## Side Effects

- Keep-alive daemon binds management port and spawns managed server.

## Errors

- Startup failure or missing status API response.

## Exit Code

0 from `Run`; assertion failures fail the test.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/config"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("unexpected run error: %s", resp.RunErr)
	}
	if resp.DaemonStatus == nil {
		t.Fatal("DaemonStatus is nil — management API did not respond")
	}
	st := resp.DaemonStatus
	if st.KeepAlivePID <= 0 {
		t.Fatalf("KeepAlivePID = %d, want > 0", st.KeepAlivePID)
	}
	if st.KeepAlivePort != config.KeepAlivePort {
		t.Fatalf("KeepAlivePort = %d, want %d", st.KeepAlivePort, config.KeepAlivePort)
	}
	if st.ServerPort != resp.ServerPort {
		t.Fatalf("ServerPort = %d, want %d", st.ServerPort, resp.ServerPort)
	}
	if !st.Running {
		t.Fatal("daemon status running=false, want true")
	}
}
```