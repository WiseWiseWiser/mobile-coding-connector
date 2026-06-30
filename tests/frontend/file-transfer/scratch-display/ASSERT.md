## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["textareaValue"]` equals `seeded-scratch-content-for-display`.
4. `GET /api/file-transfer/scratch` returns the same seeded content.

## Side Effects

- `scratch.json` remains on disk with the seeded content (no UI mutation).

## Errors

- Scratch GET on mount fails or textarea stays empty.
- Textarea shows stale or truncated content.

## Exit Code

- `0` — seeded scratch displays on load.
- `1` — server, script, or assertion failure.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v\nscript output:\n%s", err, resp.ScriptOutput)
	}

	if resp.ScriptExitCode != 0 {
		t.Fatalf("playwright script exited with code %d\noutput:\n%s", resp.ScriptExitCode, resp.ScriptOutput)
	}

	if resp.ScriptResult == nil {
		t.Fatalf("no JSON result parsed from script output:\n%s", resp.ScriptOutput)
	}

	ok, _ := resp.ScriptResult["ok"].(bool)
	if !ok {
		t.Fatalf("ScriptResult.ok is not true: %+v", resp.ScriptResult)
	}

	textareaValue, _ := resp.ScriptResult["textareaValue"].(string)
	if textareaValue != "seeded-scratch-content-for-display" {
		t.Fatalf("expected textarea %q, got %q", "seeded-scratch-content-for-display", textareaValue)
	}

	content, err := fetchScratchContent(resp.BaseURL)
	if err != nil {
		t.Fatalf("fetch scratch after display load: %v", err)
	}
	if content != "seeded-scratch-content-for-display" {
		t.Fatalf("expected API content %q, got %q", "seeded-scratch-content-for-display", content)
	}
}
```