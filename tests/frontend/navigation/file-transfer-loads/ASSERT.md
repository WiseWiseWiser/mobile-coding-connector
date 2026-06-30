## Expected

1. `Response.ServerStarted` is `true`.
2. `Response.ScriptExitCode` is `0`.
3. `Response.ScriptResult["ok"]` is `true`.
4. `Response.ScriptResult["url"]` contains `/home/file-transfer`.
5. `Response.ScriptResult["heading"]` contains `File Transfer`.
6. `Response.ScriptResult["scratchAreaVisible"]` is `true`.
7. `Response.ScriptResult["uploadAreaVisible"]` is `true`.

## Side Effects

- A quick-test server and Vite dev server are started and torn down.
- A headless Chromium browser session opens the File Transfer page.

## Errors

- Quick-test server fails to become healthy within the timeout.
- The File Transfer route is missing or the page heading/scratch/upload areas do not render.

## Exit Code

- `0` — File Transfer page loads with heading, scratch area, and upload area.
- `1` — server, script, or assertion failure.

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
	if !strings.Contains(url, "/home/file-transfer") {
		t.Fatalf("expected URL to contain /home/file-transfer, got %q", url)
	}

	heading, _ := resp.ScriptResult["heading"].(string)
	if !strings.Contains(heading, "File Transfer") {
		t.Fatalf("expected heading to contain %q, got %q", "File Transfer", heading)
	}

	scratchVisible, _ := resp.ScriptResult["scratchAreaVisible"].(bool)
	if !scratchVisible {
		t.Fatalf("expected scratch area to be visible: %+v", resp.ScriptResult)
	}

	uploadVisible, _ := resp.ScriptResult["uploadAreaVisible"].(bool)
	if !uploadVisible {
		t.Fatalf("expected upload area to be visible: %+v", resp.ScriptResult)
	}

	t.Logf("file-transfer page loaded: url=%s heading=%s scratch=%v upload=%v", url, heading, scratchVisible, uploadVisible)
}
```