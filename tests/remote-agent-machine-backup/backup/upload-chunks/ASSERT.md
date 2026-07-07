## Expected Output

Dry-run plan lists `upload-chunks` under EXCLUDED with the incomplete-upload reason;
`chunk-1` is not listed as an included file.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. EXCLUDED section mentions `upload-chunks` and `incomplete upload`.
4. Combined output does not list `.live-and-love/upload-chunks/chunk-1` as included.

## Side Effects

None (no archive write).

## Errors

- Upload-chunks content appears as included.
- Missing EXCLUDED rule for upload-chunks.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Combined, "dry-run: machine backup plan") {
		t.Fatalf("missing backup plan summary; got:\n%s", resp.Combined)
	}

	assertExcludedMentions(t, resp.Combined, "upload-chunks", "incomplete upload")
	combinedHasNone(t, resp.Combined, ".live-and-love/upload-chunks/chunk-1")
}
```