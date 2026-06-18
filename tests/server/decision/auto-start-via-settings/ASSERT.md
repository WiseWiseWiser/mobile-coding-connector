## Expected

1. `Response.ServerStarted` is `true` — the ai-critic server started and `/ping` responded
2. `Response.HasAutoStartLog` is `true` — the log contains `[opencode] AutoStartWebServer:`
3. `Response.Logs` contains:
   - `[opencode] AutoStartWebServer: BEGIN`
   - `[opencode] AutoStartWebServer: loaded settings - DefaultDomain="test-auto-start.example.com"`
   - Either:
     - `[opencode] AutoStartWebServer: attempting to start web server` (success path)
     - `[opencode] AutoStartWebServer: failed to` (failure path, e.g., no opencode binary)
4. If the opencode binary is available and the web server starts:
   - `Response.WebServerRunning` is `true`
   - Port 14096 is accessible via TCP
5. If the opencode binary is NOT available:
   - `Response.WebServerRunning` is `false`
   - The auto-start mechanism itself is still verified (log messages appeared)

## Side Effects

- An ai-critic server process is running and must be stopped during cleanup
- A temporary config home directory is created and must be removed during cleanup

## Errors

- If the server binary fails to build, the test fails
- If the server fails to start within the timeout, the test fails
- If the auto-start log messages are missing from the output, the test fails
- If Cloudflare API calls fail (no network), that is acceptable and does not
  prevent the auto-start mechanism from triggering

## Exit Code

- `0` — auto-start mechanism triggered successfully (web server may or may
  not have started depending on opencode binary availability)
- `1` — server failed to start, or auto-start log messages missing

```go
import (
	"fmt"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	if !resp.ServerStarted {
		t.Fatal("server did not start successfully")
	}

	if resp.ServerPort <= 0 {
		t.Fatal("server port not set in response")
	}

	if !resp.HasAutoStartLog {
		t.Logf("server logs:\n%s", resp.Logs)
		t.Fatal("AutoStartWebServer log messages not found in server output")
	}

	if !strings.Contains(resp.Logs, "[opencode] AutoStartWebServer: BEGIN") {
		t.Error("missing expected log: [opencode] AutoStartWebServer: BEGIN")
	}

	if !strings.Contains(resp.Logs, "AutoStartWebServer: loaded settings") {
		t.Error("missing expected log: AutoStartWebServer: loaded settings")
	}

	if req.OpenCodeSettings != nil {
		expectedDomain := req.OpenCodeSettings.DefaultDomain
		if expectedDomain != "" && !strings.Contains(resp.Logs, fmt.Sprintf("DefaultDomain=%q", expectedDomain)) {
			t.Errorf("missing expected domain %q in autostart log", expectedDomain)
		}
	}

	hasAttempt := strings.Contains(resp.Logs, "attempting to start web server")
	hasStartFail := strings.Contains(resp.Logs, "[opencode] AutoStartWebServer: failed to")

	if !hasAttempt && !hasStartFail {
		t.Error("neither 'attempting to start' nor 'failed to' found in logs; auto-start may not have reached the web server launch phase")
	}

	if resp.WebServerRunning {
		t.Logf("web server is running (opencode port accessible)")
	} else {
		t.Logf("web server is NOT running (opencode binary may be unavailable)")
		if hasAttempt {
			t.Logf("auto-start attempted to start web server but it did not become accessible")
		}
	}
}
```
