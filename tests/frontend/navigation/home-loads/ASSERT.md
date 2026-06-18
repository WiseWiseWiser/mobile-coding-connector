## Expected

1. `Response.ServerStarted` is `true`.
2. `Response.ScriptExitCode` is `0`.
3. `Response.ScriptResult["ok"]` is `true`.
4. `Response.ScriptResult["url"]` contains `/home`.
5. `Response.ScriptResult["heading"]` is `Your Projects`.

## Side Effects

- A quick-test server and Vite dev server are started and torn down.
- A headless Chromium browser session is opened via `playwright-debug`.

## Errors

- Quick-test server fails to become healthy within the timeout.
- `playwright-debug` is missing or the script throws.
- Workspace list UI does not render.

## Exit Code

- `0` — home page loads and workspace UI is present.
- `1` — server, script, or assertion failure.
```

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v\nscript output:\n%s", err, resp.ScriptOutput)
	}

	if !resp.ServerStarted {
		t.Fatal("quick-test server did not start successfully")
	}

	if resp.ScriptExitCode != 0 {
		t.Fatalf("playwright script exited with code %d\noutput:\n%s", resp.ScriptExitCode, resp.ScriptOutput)
	}

	if resp.ScriptResult == nil {
		t.Fatalf("no JSON result parsed from script output:\n%s", resp.ScriptOutput)
	}

	ok, _ := resp.ScriptResult["ok"].(bool)
	if !ok {
		t.Fatalf("ScriptResult.ok is not true: %+v\noutput:\n%s", resp.ScriptResult, resp.ScriptOutput)
	}

	url, _ := resp.ScriptResult["url"].(string)
	if !strings.Contains(url, "/home") {
		t.Fatalf("expected URL to contain /home, got %q", url)
	}

	heading, _ := resp.ScriptResult["heading"].(string)
	if heading != "Your Projects" {
		t.Fatalf("expected heading %q, got %q", "Your Projects", heading)
	}

	if req.Headless == nil || *req.Headless {
		t.Logf("ran headless (shell mode)")
	} else {
		t.Fatalf("expected headless mode by default, got visible browser")
	}

	t.Logf("home page loaded: url=%s heading=%s baseURL=%s", url, heading, resp.BaseURL)
}
```