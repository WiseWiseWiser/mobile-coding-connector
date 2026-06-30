## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["scratchAreaVisible"]` is `true`.
4. `Response.ScriptResult["textareaEmpty"]` is `true`.
5. `GET /api/file-transfer/scratch` returns `content: ""`.

## Side Effects

- No `scratch.json` file exists under `{AI_CRITIC_HOME}/file-transfer/`.

## Errors

- Scratch section or textarea `data-testid` selectors are missing.
- Textarea is pre-filled when no scratch file exists.
- Scratch API endpoint is not registered or returns non-empty content.

## Exit Code

- `0` — empty scratch pad renders and API reports empty content.
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

	scratchVisible, _ := resp.ScriptResult["scratchAreaVisible"].(bool)
	if !scratchVisible {
		t.Fatal("expected scratch area to be visible")
	}

	textareaEmpty, _ := resp.ScriptResult["textareaEmpty"].(bool)
	if !textareaEmpty {
		t.Fatalf("expected empty textarea, got %+v", resp.ScriptResult)
	}

	content, err := fetchScratchContent(resp.BaseURL)
	if err != nil {
		t.Fatalf("fetch scratch after empty load: %v", err)
	}
	if content != "" {
		t.Fatalf("expected API content empty string, got %q", content)
	}
}
```