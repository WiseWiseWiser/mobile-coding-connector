## Expected

1. `Response.ServerStarted` is `true` — the ai-critic server started and `/ping` responded
2. `Response.HasAutoStartLog` is `false` — the log must NOT contain `[opencode] AutoStartWebServer:`
3. `Response.PrePostLogs` is empty — no post-start action was performed
4. `Response.WebServerRunning` is `false` — port 14096 should NOT be accessible

## Side Effects

- An ai-critic server process is running and must be stopped during cleanup
- A temporary config home directory is created and must be removed during cleanup

## Errors

- If the server binary fails to build, the test fails
- If the server fails to start within the timeout, the test fails
- If auto-start log messages ARE present in the output, the test fails

## Exit Code

- `0` — server started and auto-start was correctly NOT triggered
- `1` — server failed to start, or auto-start was incorrectly triggered
```

```go
import (
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

	if resp.HasAutoStartLog {
		t.Logf("server logs:\n%s", resp.Logs)
		t.Fatal("AutoStartWebServer log messages found in server output, but should NOT be present when WebServer.Enabled=false")
	}

	if strings.Contains(resp.Logs, "[opencode] AutoStartWebServer:") {
		t.Error("found auto-start log prefix in output, should be absent")
	}

	if resp.WebServerRunning {
		t.Error("web server port is accessible, should NOT be when auto-start is disabled")
	}

	if resp.PrePostLogs != "" {
		t.Error("PrePostLogs should be empty for non-post-start tests")
	}

	if t.Failed() {
		t.Logf("server logs:\n%s", resp.Logs)
	}
}
```
