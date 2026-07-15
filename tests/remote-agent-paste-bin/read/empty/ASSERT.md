## Expected

1. Exit code 0.
2. Stdout is empty (silent).
3. Stderr is empty by default.

## Side Effects

Scratch API still reports `content: ""` (missing file treated as empty).

## Errors

- Any stdout/stderr output on empty scratch read.
- Non-zero exit.

## Exit Code

0.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	assertStdoutEmpty(t, resp.Stdout)
	if resp.Stderr != "" {
		t.Fatalf("expected silent stderr; got:\n%s", resp.Stderr)
	}
	assertScratchContentEmpty(t, resp.ScratchAfter.Content)
}
```