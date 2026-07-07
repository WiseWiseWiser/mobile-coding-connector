## Expected Output

Stdout begins with `home: <serverHome>`, prints `> plain-dir` and `> notes.txt`
entry blocks, and ends with `analyse-files summary`.

## Expected

1. Exit code 0.
2. Combined output contains `home:` referencing the harness server home path.
3. Combined output contains `analyse-files summary`.
4. Combined output contains top-level entry headers `> plain-dir` and `> notes.txt`.
5. `plain-dir` block lists at least one immediate child line starting with `> sub`.

## Side Effects

None (read-only scan).

## Errors

- Non-zero exit.
- Missing `home:` line, entry headers, or summary block.

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

	combinedHasAll(t, resp.Combined,
		"home:",
		"analyse-files summary",
		"> plain-dir",
		"> notes.txt",
	)

	if !strings.Contains(resp.Combined, resp.ServerHome) &&
		!strings.Contains(resp.Combined, "home:") {
		t.Fatalf("home line should reference server home %q; got:\n%s", resp.ServerHome, resp.Combined)
	}

	plainBlock := extractEntryBlock(t, resp.Combined, "plain-dir")
	if !strings.Contains(plainBlock, "> sub") {
		t.Fatalf("plain-dir block missing child > sub:\n%s", plainBlock)
	}
}
```