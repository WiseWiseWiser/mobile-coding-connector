## Expected

1. `Run` completes without error.
2. `OccupierServerKilled` and `OccupierDaemonKilled` are both true.
3. `DaemonStatus` is non-nil with `KeepAlivePID > 0` and `Running` true.
4. `DaemonStatus.ServerPort` matches `Response.ServerPort`.

## Side Effects

- Both stale listeners removed before daemon startup.

## Errors

- Either occupier survives or daemon fails to reach running state.

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
	if !resp.OccupierServerKilled {
		t.Fatal("server port occupier still alive")
	}
	if !resp.OccupierDaemonKilled {
		t.Fatal("daemon port occupier still alive")
	}
	if resp.DaemonStatus == nil || resp.DaemonStatus.KeepAlivePID <= 0 {
		t.Fatal("daemon did not start after killing both occupiers")
	}
	if !resp.DaemonStatus.Running {
		t.Fatal("daemon status running=false after killing both occupiers")
	}
	if resp.DaemonStatus.ServerPort != resp.ServerPort {
		t.Fatalf("ServerPort = %d, want %d", resp.DaemonStatus.ServerPort, resp.ServerPort)
	}
}
```