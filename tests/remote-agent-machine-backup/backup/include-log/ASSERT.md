## Expected Output

Dry-run plan lists `.ai-critic/keep.log` under DOT FILES; `.ai-critic/service.log`
remains excluded by the log suffix rule.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. DOT FILES includes `.ai-critic/keep.log`.
4. DOT FILES does not include `.ai-critic/service.log`.

## Side Effects

None.

## Errors

- Re-included log still excluded.
- Unrelated log incorrectly included.

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

	assertDotFilesIncludes(t, resp.Combined, ".ai-critic/keep.log")
	assertDotFilesExcludes(t, resp.Combined, ".ai-critic/service.log")
}
```