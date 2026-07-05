## Expected

1. `RestartHTTPStatus` is `200`.
2. `RestartJSONStatus` is `restart_requested`.
3. `AfterServerPID` is positive and differs from `BeforeServerPID`.
4. `AfterDaemonPID` equals `BeforeDaemonPID` (daemon not exec-replaced).
5. `DaemonReachable` is true.
6. `ServerPingOK` is true after settle.

## Side Effects

- Managed server process replaced; keep-alive daemon process unchanged.

## Errors

- Status not `restart_requested`, server PID unchanged, or daemon unreachable.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.RestartHTTPStatus != 200 {
		t.Fatalf("POST /restart status = %d, want 200", resp.RestartHTTPStatus)
	}
	if resp.RestartJSONStatus != "restart_requested" {
		t.Fatalf("restart JSON status = %q, want restart_requested", resp.RestartJSONStatus)
	}
	if resp.BeforeServerPID <= 0 || resp.AfterServerPID <= 0 {
		t.Fatalf("server PID missing: before=%d after=%d", resp.BeforeServerPID, resp.AfterServerPID)
	}
	if resp.AfterServerPID == resp.BeforeServerPID {
		t.Fatalf("server PID unchanged: %d", resp.BeforeServerPID)
	}
	if resp.AfterDaemonPID != resp.BeforeDaemonPID {
		t.Fatalf("daemon PID changed: before=%d after=%d (signal restart must not exec daemon)",
			resp.BeforeDaemonPID, resp.AfterDaemonPID)
	}
	if !resp.DaemonReachable {
		t.Fatal("daemon status API unreachable after server restart")
	}
	if !resp.ServerPingOK {
		t.Fatal("GET /ping failed after server restart")
	}
}
```