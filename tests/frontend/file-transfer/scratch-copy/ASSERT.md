## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["clipboardText"]` equals `seeded-scratch-for-copy-test`.

## Side Effects

- Clipboard contains the seeded scratch text after Copy (browser session only).
- `scratch.json` on disk is unchanged.

## Errors

- Copy button missing or `navigator.clipboard.writeText` fails.
- Clipboard read permission denied in headless Chromium.
- Clipboard text does not match seeded scratch content.

## Exit Code

- `0` — Copy places seeded content on the clipboard.
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

	clipboardText, _ := resp.ScriptResult["clipboardText"].(string)
	if clipboardText != "seeded-scratch-for-copy-test" {
		t.Fatalf("expected clipboard %q, got %q", "seeded-scratch-for-copy-test", clipboardText)
	}
}
```