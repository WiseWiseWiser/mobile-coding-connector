## Expected Output

Dry-run plan lists `**(binary)` under EXCLUDED with executable-binary reason;
`.ai-critic/bin/stub` is not listed in DOT FILES.

## Expected

1. Exit code 0.
2. Combined output contains `dry-run: machine backup plan`.
3. EXCLUDED section mentions `(binary)` and `executable`.
4. DOT FILES does not include `.ai-critic/bin/stub`.

## Side Effects

None.

## Errors

- ELF stub listed as included.
- Missing binary exclusion rule in EXCLUDED.

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

	assertExcludedMentions(t, resp.Combined, "(binary)", "executable")
	assertDotFilesExcludes(t, resp.Combined, ".ai-critic/bin/stub")
}
```