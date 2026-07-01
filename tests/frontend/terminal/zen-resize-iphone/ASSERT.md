## Expected

1. `Response.ServerStarted` is `true`.
2. `Response.ScriptExitCode` is `0`.
3. `Response.ScriptResult["ok"]` is `true`.
4. The page remains on `/project/{name}/terminal`.
5. At least one initial terminal resize is captured before zen mode.
6. Entering zen mode sends at least one additional resize JSON message.
7. Exiting zen mode sends at least one further resize JSON message.
8. The latest resize payload has positive numeric `cols` and `rows`.

## Side Effects

- A temporary project record is created in the isolated quick-test config home.
- A terminal session may be created by `/api/terminal` and cleaned up by the browser closing without user input.

## Errors

- Project terminal route fails to load on an iPhone-sized viewport.
- The active terminal tab never connects or never sends the initial resize.
- `Zen` or `Exit Zen` changes visible state without sending a backend resize.
- Resize JSON is malformed or contains non-positive dimensions.

## Exit Code

- `0` — zen entry and exit both emit positive terminal resize messages.
- `1` — server, script, or assertion failure.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		output := ""
		if resp != nil {
			output = resp.ScriptOutput
		}
		t.Fatalf("Run returned unexpected error: %v\nscript output:\n%s", err, output)
	}

	if resp == nil {
		t.Fatal("Run returned nil response")
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
		t.Fatalf("expected zen entry and exit to send resize messages, got result: %+v\noutput:\n%s", resp.ScriptResult, resp.ScriptOutput)
	}

	url, _ := resp.ScriptResult["url"].(string)
	if !strings.Contains(url, "/project/") || !strings.Contains(url, "/terminal") {
		t.Fatalf("expected project terminal URL, got %q", url)
	}

	if count := numberField(resp.ScriptResult, "initialResizeCount"); count < 1 {
		t.Fatalf("expected at least one initial resize before zen, got %v in %+v", count, resp.ScriptResult)
	}
	if count := numberField(resp.ScriptResult, "entryResizeCount"); count < 1 {
		t.Fatalf("expected at least one resize after zen entry, got %v in %+v", count, resp.ScriptResult)
	}
	if count := numberField(resp.ScriptResult, "exitResizeCount"); count < 1 {
		t.Fatalf("expected at least one resize after zen exit, got %v in %+v", count, resp.ScriptResult)
	}

	lastResize, _ := resp.ScriptResult["lastResize"].(map[string]any)
	if lastResize == nil {
		t.Fatalf("expected lastResize object, got %+v", resp.ScriptResult["lastResize"])
	}
	cols := numberField(lastResize, "cols")
	rows := numberField(lastResize, "rows")
	if cols <= 0 || rows <= 0 {
		t.Fatalf("expected positive last resize dimensions, got cols=%v rows=%v payload=%+v", cols, rows, lastResize)
	}

	if req.TimeoutSecs < 150 {
		t.Fatalf("expected timeout to allow terminal startup, got %d", req.TimeoutSecs)
	}
}

func numberField(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
```
