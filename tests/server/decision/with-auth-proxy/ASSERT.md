## Expected

1. `Response.ServerStarted` is `true` — the ai-critic server started and `/ping` responded
2. `Response.HasAutoStartLog` is `true` — the log contains `[opencode] AutoStartWebServer:`
3. `Response.Logs` contains:
   - `[opencode] AutoStartWebServer: BEGIN`
   - `[opencode] AutoStartWebServer: loaded settings` with `AuthProxyEnabled=true`
   - Either:
     - `[basic_auth_proxy] Proxy started on port` (success path)
     - `StartWebServer returned error` or `failed to` (failure path, e.g., no opencode binary)
4. If the opencode binary is available and the proxy starts:
   - `Response.ProxyRunning` is `true` — proxy port 14100 is accessible via TCP
   - `Response.BackendRunning` is `true` — backend port is accessible via TCP
   - `Response.BackendPort > 0` — backend port was read from `basic-auth-proxy.json`
   - `Response.Logs` contains `[basic_auth_proxy] Proxy started on port` with port and backend
5. If the opencode binary is NOT available:
   - `Response.ProxyRunning` may be `false`
   - The auto-start mechanism itself is still verified (log messages appeared)
   - The test passes as long as auto-start was triggered

## Side Effects

- An ai-critic server process is running and must be stopped during cleanup
- A basic-auth-proxy subprocess may be running on port 14100
- A temporary config home directory is created and must be removed during cleanup
- A `basic-auth-proxy.json` file is written to the config home (read by the test)

## Errors

- If the server binary fails to build, the test fails
- If the basic-auth-proxy binary fails to build, the test fails
- If the server fails to start within the timeout, the test fails
- If the auto-start log messages are missing from the output, the test fails
- If the proxy auth start message is not found when the binary is available, the test logs a warning
- If Cloudflare API calls fail (no network), that is acceptable and does not
  prevent the auto-start mechanism from triggering

## Exit Code

- `0` — auto-start mechanism triggered successfully with auth proxy (web server and proxy
  may or may not have started depending on opencode binary availability)
- `1` — server failed to start, or auto-start log messages missing
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

	if !strings.Contains(resp.Logs, "AuthProxyEnabled=true") {
		t.Error("missing expected log field: AuthProxyEnabled=true in autostart settings")
	}

	if req.OpenCodeSettings != nil {
		expectedDomain := req.OpenCodeSettings.DefaultDomain
		if expectedDomain != "" && !strings.Contains(resp.Logs, expectedDomain) {
			t.Errorf("missing expected domain %q in server logs", expectedDomain)
		}
	}

	hasProxyLog := strings.Contains(resp.Logs, "[basic_auth_proxy] Proxy started on port")
	hasStartFail := strings.Contains(resp.Logs, "StartWebServer returned error") ||
		strings.Contains(resp.Logs, "[opencode] AutoStartWebServer: failed to")

	if hasProxyLog {
		t.Logf("auth proxy start message found in logs")
	} else if hasStartFail {
		t.Logf("web server/proxy start failure detected (opencode binary may be unavailable)")
	} else {
		t.Logf("neither proxy start nor failure message found in logs")
	}

	if resp.ProxyRunning {
		t.Logf("proxy port is reachable (proxy is running)")
	} else {
		t.Logf("proxy port is NOT reachable (opencode binary may be unavailable)")
		if hasProxyLog {
			t.Errorf("proxy start message found but proxy port is not reachable")
		}
	}

	if resp.BackendRunning {
		t.Logf("backend port %d is reachable (opencode web server is running)", resp.BackendPort)
	} else {
		t.Logf("backend port is NOT reachable (opencode binary may be unavailable)")
	}

	if resp.BackendPort > 0 {
		t.Logf("backend port discovered from basic-auth-proxy.json: %d", resp.BackendPort)
	}

	if hasProxyLog && resp.BackendPort <= 0 {
		t.Errorf("proxy start message found but backend port was not discovered")
	}

	if t.Failed() {
		t.Logf("=== FULL SERVER OUTPUT ===\n%s\n=== END ===", resp.Logs)
	}
}
```
