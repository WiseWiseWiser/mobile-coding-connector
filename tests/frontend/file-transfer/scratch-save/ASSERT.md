## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["textareaValue"]` equals `saved-from-playwright-scratch-test`.
4. `GET /api/file-transfer/scratch` returns `saved-from-playwright-scratch-test`.

## Side Effects

- `{AI_CRITIC_HOME}/file-transfer/scratch.json` is created or updated on disk.

## Errors

- Save button missing or not wired to PUT.
- Textarea clears after save or API returns different content.
- Save fails without surfacing an error in the UI.

## Exit Code

- `0` — Save persists content in UI and API.
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
	if textareaValue != "saved-from-playwright-scratch-test" {
		t.Fatalf("expected textarea %q after save, got %q", "saved-from-playwright-scratch-test", textareaValue)
	}

	content, err := fetchScratchContent(resp.BaseURL)
	if err != nil {
		t.Fatalf("fetch scratch after save: %v", err)
	}
	if content != "saved-from-playwright-scratch-test" {
		t.Fatalf("expected API content %q after save, got %q", "saved-from-playwright-scratch-test", content)
	}
}
```