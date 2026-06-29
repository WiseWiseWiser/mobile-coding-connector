## Expected

1. `Response.ServerReady` is true.
2. `Response.CoreReadyMs` is between 0 and 3000 inclusive when bootstrap marker present;
   if marker absent (pre-implementation), `/ping` must succeed within 3s of first
   `Starting ai-critic server` log line (fallback: `PortReadyMs` <= 3000 when parsed).
3. `Response.RestartLoopSeen` is false.
4. Merged logs must not contain `[opencode] AutoStartWebServer: BEGIN` (no extension tunnel path).

## Side Effects

- Minimal config home with disabled web server settings.

## Errors

- Slow core or restart loop fails test.

## Exit Code

- `0` on fast stable core start.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ServerReady {
		t.Fatal("server not ready")
	}
	if resp.RestartLoopSeen {
		t.Fatal("unexpected restart loop on baseline")
	}
	if resp.CoreReadyMs >= 0 && resp.CoreReadyMs > 3000 {
		t.Fatalf("core_ready t_ms=%d exceeds 3000ms budget", resp.CoreReadyMs)
	}
	if resp.PortReadyMs >= 0 && resp.PortReadyMs > 3000 && resp.CoreReadyMs < 0 {
		t.Fatalf("daemon waited_ms=%d exceeds 3000ms without core_ready marker", resp.PortReadyMs)
	}
	combined := resp.DaemonLogs + resp.ServerLogs
	if strings.Contains(combined, "[opencode] AutoStartWebServer: BEGIN") {
		t.Fatal("extension autostart ran on baseline no-extension path")
	}
}
```