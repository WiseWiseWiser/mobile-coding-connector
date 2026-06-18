## Expected

1. `Response.ServerStarted` is `true` — the ai-critic server started and `/ping` responded
2. `Response.PrePostLogs` is non-empty and does NOT contain `[opencode] AutoStartWebServer:` — no auto-start on initial boot with `Enabled=false`
3. `Response.Logs` (full output including post-API call) DOES contain `[opencode] AutoStartWebServer:` — auto-start was triggered after the settings API call enabled the web server
4. `Response.HasAutoStartLog` is `true` — the full log contains auto-start messages
5. The initial startup segment (`PrePostLogs`) must NOT have auto-start messages
6. At least one auto-start log message should appear after the API call point
7. If the opencode binary is available: `Response.WebServerRunning` is `true`, port 14096 is accessible

## Side Effects

- An ai-critic server process is running and must be stopped during cleanup
- A temporary config home directory is created and must be removed during cleanup
- The settings API call modifies `opencode.json` on disk (enables the web server)

## Errors

- If the server binary fails to build, the test fails
- If the server fails to start within the timeout, the test fails
- If auto-start log messages appear in the initial startup segment, the test fails
- If no auto-start log messages appear in the full output after the API call, the test fails (bug not fixed)

## Exit Code

- `0` — server started, initial logs had no auto-start, API call triggered auto-start
- `1` — server failed to start, or auto-start messages in wrong segment, or auto-start never triggered
```

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

	if resp.PrePostLogs == "" {
		t.Fatal("PrePostLogs is empty; post-start snapshot was not captured")
	}

	if strings.Contains(resp.PrePostLogs, "[opencode] AutoStartWebServer:") {
		t.Logf("PrePostLogs (initial startup):\n%s", resp.PrePostLogs)
		t.Fatal("PrePostLogs contains auto-start messages, but should NOT since WebServer.Enabled=false at boot")
	}

	if !resp.HasAutoStartLog {
		t.Logf("full server logs:\n%s", resp.Logs)
		t.Logf("PrePostLogs (initial startup):\n%s", resp.PrePostLogs)
		t.Fatal("AutoStartWebServer log messages not found in full server output after API call")
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

	t.Logf("Initial startup logs (before API call) — verified no auto-start: %d bytes", len(resp.PrePostLogs))
	t.Logf("Full logs (after API call) — verified auto-start triggered: %d bytes", len(resp.Logs))

	if resp.WebServerRunning {
		t.Logf("web server is running (opencode port accessible)")
	} else {
		t.Logf("web server is NOT running (opencode binary may be unavailable)")
	}

	if t.Failed() {
		t.Logf("=== FULL SERVER OUTPUT ===\n%s\n=== END ===", resp.Logs)
	}
}
```
