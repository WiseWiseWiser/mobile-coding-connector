## Expected

1. Exit code 0.
2. Stdout equals seeded content byte-for-byte (including newline and emoji).
3. Stderr empty by default.

## Side Effects

Scratch API content unchanged.

## Errors

- Truncation, extra newlines, or encoding corruption on stdout.
- Unexpected stderr on default read.

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
	assertStdoutExactBytes(t, resp.Stdout, []byte(seededUTF8Content))
	if resp.Stderr != "" {
		t.Fatalf("expected silent stderr; got:\n%s", resp.Stderr)
	}
	assertScratchContentExact(t, resp.ScratchAfter.Content, seededUTF8Content)
}
```