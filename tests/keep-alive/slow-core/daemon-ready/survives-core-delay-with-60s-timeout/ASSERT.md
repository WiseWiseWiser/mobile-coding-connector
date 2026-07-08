## Expected

1. `Response.ServerReady` is true — daemon logged ready or `[keepalive] phase=server_ready`.
2. `Response.RestartLoopSeen` is false — no `failed to become ready` in merged logs.
3. `Response.PortReadyMs` is between 15000 and 60000 inclusive when parsed; if marker
   absent (pre-implementation), independent `/ping` must succeed before observation ends
   without prior restart-loop messages.

## Side Effects

- Longer (65s) daemon observation per test.

## Errors

- Daemon fails to start.

## Exit Code

- `0` when server becomes ready within 60s startup timeout despite 15s core delay.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ServerReady {
		t.Fatalf("daemon did not observe server ready within 60s; logs tail in test output")
	}
	if resp.RestartLoopSeen {
		t.Fatal("restart loop detected: server failed to become ready within timeout")
	}
	if resp.PortReadyMs >= 0 {
		if resp.PortReadyMs < 15000 {
			t.Fatalf("waited_ms=%d is below 15s core delay — hook may not be applied", resp.PortReadyMs)
		}
		if resp.PortReadyMs > 60000 {
			t.Fatalf("waited_ms=%d exceeds 60s startup timeout", resp.PortReadyMs)
		}
	}
}
```