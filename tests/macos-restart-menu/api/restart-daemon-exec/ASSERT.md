---
label: slow
explanation: daemon exec restart and settle polling
---

## Expected

1. `RestartHTTPStatus` is `200`.
2. `SSEDoneSuccess` is true (SSE `done` event with `success=true`).
3. `DaemonReachable` is true after settle.
4. `AfterServerPID` is positive (managed server running again).
5. `ServerPingOK` is true after settle.

## Side Effects

- Keep-alive daemon exec-replaces itself (same PID via `syscall.Exec`); managed server
  respawns; management API on `23312` responds again.

## Errors

- Missing SSE done success, daemon status unavailable, or `/ping` fails after restart.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.RestartHTTPStatus != 200 {
		t.Fatalf("POST /restart-daemon status = %d, want 200; body=%q", resp.RestartHTTPStatus, resp.SSEBody)
	}
	if !resp.SSEDoneSuccess {
		t.Fatalf("SSE done.success not true; body=%q", resp.SSEBody)
	}
	if !resp.DaemonReachable {
		t.Fatal("daemon status API unreachable after exec restart")
	}
	if resp.AfterServerPID <= 0 {
		t.Fatalf("server not running after daemon restart; before_server_pid=%d", resp.BeforeServerPID)
	}
	if !resp.ServerPingOK {
		t.Fatal("GET /ping failed after daemon restart")
	}
}
```