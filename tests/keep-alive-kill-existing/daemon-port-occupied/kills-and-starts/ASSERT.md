## Expected

1. `Run` completes without error.
2. `OccupierDaemonPID` was greater than zero.
3. `OccupierDaemonKilled` is true.
4. `DaemonStatus` is non-nil; `GET /api/keep-alive/status` succeeded.
5. `DaemonStatus.KeepAlivePort` is `23312`.

## Side Effects

- Prior management-port listener terminated; new daemon serves status API.

## Errors

- Occupier survives or status API unreachable.

## Exit Code

0 from `Run`.

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
	if resp.OccupierDaemonPID <= 0 {
		t.Fatal("daemon port occupier PID missing")
	}
	if !resp.OccupierDaemonKilled {
		t.Fatalf("daemon occupier pid=%d still alive after --kill-existing", resp.OccupierDaemonPID)
	}
	if resp.DaemonStatus == nil {
		t.Fatal("status API did not respond after killing daemon port occupier")
	}
	if resp.DaemonStatus.KeepAlivePort != config.KeepAlivePort {
		t.Fatalf("KeepAlivePort = %d, want %d", resp.DaemonStatus.KeepAlivePort, config.KeepAlivePort)
	}
}
```