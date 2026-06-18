## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["emptyStateVisible"]` is `true`.
4. `Response.ScriptResult["fileCount"]` equals `0`.

## Side Effects

- `{AI_CRITIC_HOME}/file-transfer/` exists and contains no user files.

## Errors

- Empty-state message is missing when the directory has no files.
- Stray file rows appear in the list.

## Exit Code

- `0` — empty inbox renders correctly.
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

	emptyVisible, _ := resp.ScriptResult["emptyStateVisible"].(bool)
	if !emptyVisible {
		t.Fatal("expected empty-state message to be visible")
	}

	fileCount, _ := resp.ScriptResult["fileCount"].(float64)
	if int(fileCount) != 0 {
		t.Fatalf("expected fileCount 0, got %v", fileCount)
	}
}
```