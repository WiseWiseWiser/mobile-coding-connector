## Expected

1. `Response.ServerStarted` is `true`.
2. `Response.ScriptExitCode` is `0`.
3. `Response.ScriptResult["ok"]` is `true`.
4. `Response.ScriptResult["url"]` ends with `/home` or contains `/home`.

## Side Effects

- A quick-test server and Vite dev server are started and torn down.

## Errors

- Quick-test server fails to become healthy within the timeout.
- Root path does not redirect to `/home`.

## Exit Code

- `0` — root redirects to home.
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
	if !strings.HasSuffix(url, "/home") && !strings.Contains(url, "/home") {
		t.Fatalf("expected final URL to end with or contain /home, got %q", url)
	}

	t.Logf("root redirect ok: final url=%s", url)
}
```