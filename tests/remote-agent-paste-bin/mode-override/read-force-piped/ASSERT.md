## Expected

1. Exit code 0.
2. Stdout equals seeded scratch content (not piped junk).
3. Scratch API `content` still equals seeded value.
4. Stderr has no `saved N bytes` write summary.

## Side Effects

Scratch unchanged despite piped stdin.

## Errors

- Piped junk written to scratch API.
- Write-mode stderr on forced read.
- Stdout shows piped junk instead of seed.

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

	assertStdoutExactBytes(t, resp.Stdout, []byte(forceReadSeedContent))
	assertScratchContentExact(t, resp.ScratchAfter.Content, forceReadSeedContent)

	if strings.Contains(resp.Stderr, "saved ") {
		t.Fatalf("--read must not run write path; stderr:\n%s", resp.Stderr)
	}
	if resp.Stdout == forceReadIgnoredPipe {
		t.Fatalf("piped stdin must be ignored when --read is set")
	}
}
```