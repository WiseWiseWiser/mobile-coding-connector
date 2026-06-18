## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["downloadedName"]` is `hello.txt`.

## Side Effects

- Browser download event fires for `hello.txt`.
- `hello.txt` remains in `{AI_CRITIC_HOME}/file-transfer/` after download.

## Errors

- Download button missing or does not trigger a browser download.
- Suggested filename differs from stored name.

## Exit Code

- `0` — download succeeds with correct filename.
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

	downloadedName, _ := resp.ScriptResult["downloadedName"].(string)
	if downloadedName != "hello.txt" {
		t.Fatalf("expected downloadedName hello.txt, got %q", downloadedName)
	}
}
```