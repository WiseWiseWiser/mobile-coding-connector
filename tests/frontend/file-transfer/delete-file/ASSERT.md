## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["tempVisible"]` is `false`.
4. `GET /api/file-transfer` does not include `temp.txt`.

## Side Effects

- `temp.txt` is removed from `{AI_CRITIC_HOME}/file-transfer/` on disk.

## Errors

- Confirmation dialog not shown or not accepted.
- Row remains visible after confirm.
- API list still contains `temp.txt`.

## Exit Code

- `0` — delete succeeds in UI and API.
- `1` — server, script, or assertion failure.

```go
import (
	"slices"
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

	tempVisible, _ := resp.ScriptResult["tempVisible"].(bool)
	if tempVisible {
		t.Fatal("expected temp.txt row to disappear after delete")
	}

	names, err := fetchFileTransferNames(resp.BaseURL)
	if err != nil {
		t.Fatalf("fetch file-transfer list after delete: %v", err)
	}
	if slices.Contains(names, "temp.txt") {
		t.Fatalf("expected temp.txt absent from API list, got %v", names)
	}
}
```