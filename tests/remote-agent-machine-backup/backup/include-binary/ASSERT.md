## Expected Output

Dry-run plan lists `.ai-critic/bin/stub` under DOT FILES after `--include`.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. DOT FILES includes `.ai-critic/bin/stub`.

## Side Effects

None.

## Errors

- Re-included executable still excluded from DOT FILES.

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

	assertDotFilesIncludes(t, resp.Combined, ".ai-critic/bin/stub")
}
```