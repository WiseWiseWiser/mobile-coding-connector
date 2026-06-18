## Expected

1. `Response.ServerStarted` is `true`.
2. `Response.ScriptExitCode` is `0`.
3. `Response.ScriptResult["ok"]` is `true`.
4. `Response.ScriptResult["url"]` contains `/home/tools`.
5. Either `Response.ScriptResult["heading"]` contains `Server Tools` or `Response.ScriptResult["foundationVisible"]` is `true`.

## Side Effects

- A quick-test server and Vite dev server are started and torn down.
- The tools check SSE endpoint is exercised.

## Errors

- Quick-test server fails to become healthy within the timeout.
- Tools page heading and Foundation category are both absent.

## Exit Code

- `0` — tools page loads with expected UI markers.
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
	if !strings.Contains(url, "/home/tools") {
		t.Fatalf("expected URL to contain /home/tools, got %q", url)
	}

	heading, _ := resp.ScriptResult["heading"].(string)
	foundationVisible, _ := resp.ScriptResult["foundationVisible"].(bool)
	if !strings.Contains(heading, "Server Tools") && !foundationVisible {
		t.Fatalf("expected Server Tools heading or Foundation category, got heading=%q foundationVisible=%v", heading, foundationVisible)
	}

	t.Logf("tools page loaded: url=%s heading=%q foundationVisible=%v", url, heading, foundationVisible)
}
```