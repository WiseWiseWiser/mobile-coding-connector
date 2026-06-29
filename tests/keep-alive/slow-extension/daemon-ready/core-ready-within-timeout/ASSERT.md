## Expected

1. `Response.ServerReady` is true — daemon logged ready or `[keepalive] phase=server_ready`.
2. `Response.PortReadyMs` is between 0 and 10000 inclusive when parsed; otherwise
   daemon log contains `Server is ready` without prior `failed to become ready`.
3. `Response.RestartLoopSeen` is false.
4. Independent `/ping` succeeded during observation (`PingBeforeExt` may be true once extension logs exist).

## Side Effects

- Keep-alive daemon and managed server processes run in isolated temp config home.

## Errors

- Build or daemon start failure fails the test immediately.

## Exit Code

- `0` when daemon observes ready within 10s startup timeout semantics.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ServerReady {
		t.Fatalf("daemon did not observe server ready; logs tail in test output")
	}
	if resp.RestartLoopSeen {
		t.Fatal("restart loop detected: server failed to become ready within timeout")
	}
	if resp.PortReadyMs >= 0 && resp.PortReadyMs > 10000 {
		t.Fatalf("waited_ms=%d exceeds 10s startup timeout", resp.PortReadyMs)
	}
}
```