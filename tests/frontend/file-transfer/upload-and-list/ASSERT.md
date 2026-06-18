## Expected

1. `Response.ScriptExitCode` is `0`.
2. `Response.ScriptResult["ok"]` is `true`.
3. `Response.ScriptResult["fileName"]` is `sample.txt`.
4. `Response.ScriptResult["fileCount"]` is at least `1`.
5. `Response.ScriptResult["rowText"]` contains `sample.txt`.

## Side Effects

- `sample.txt` is written to `{AI_CRITIC_HOME}/file-transfer/sample.txt` on disk.
- The file appears in `GET /api/file-transfer` list (newest first).

## Errors

- Upload UI missing or file input not wired.
- Row does not appear after upload completes.
- Server rejects upload (e.g., size limit) without surfacing an error in the UI.

## Exit Code

- `0` — upload succeeds and list shows the new file.
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

	fileName, _ := resp.ScriptResult["fileName"].(string)
	if fileName != "sample.txt" {
		t.Fatalf("expected fileName sample.txt, got %q", fileName)
	}

	fileCount, _ := resp.ScriptResult["fileCount"].(float64)
	if int(fileCount) < 1 {
		t.Fatalf("expected at least one file row, got %v", fileCount)
	}

	rowText, _ := resp.ScriptResult["rowText"].(string)
	if !strings.Contains(rowText, "sample.txt") {
		t.Fatalf("expected row text to contain sample.txt, got %q", rowText)
	}
}
```