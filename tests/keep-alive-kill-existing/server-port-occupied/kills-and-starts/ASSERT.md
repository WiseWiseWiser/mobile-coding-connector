## Expected

1. `Run` completes without error.
2. `OccupierServerPID` was greater than zero at start.
3. `OccupierServerKilled` is true.
4. `DaemonStatus` is non-nil with `KeepAlivePID > 0`.
5. `DaemonStatus.Running` is true.

## Side Effects

- Prior server-port listener terminated before managed server spawn.

## Errors

- Occupier survives or daemon fails to start.

## Exit Code

0 from `Run`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("unexpected run error: %s", resp.RunErr)
	}
	if resp.OccupierServerPID <= 0 {
		t.Fatal("server port occupier PID missing")
	}
	if !resp.OccupierServerKilled {
		t.Fatalf("server occupier pid=%d still alive after --kill-existing", resp.OccupierServerPID)
	}
	if resp.DaemonStatus == nil || resp.DaemonStatus.KeepAlivePID <= 0 {
		t.Fatal("daemon did not start after killing server port occupier")
	}
	if !resp.DaemonStatus.Running {
		t.Fatal("daemon status running=false after kill-existing on server port")
	}
}
```