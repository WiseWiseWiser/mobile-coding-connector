## Expected

1. Non-zero exit (typically 1).
2. Combined output mentions unexpected/extra arguments or usage/help for `paste-bin`.
3. Scratch API content unchanged from seed.

## Side Effects

No scratch mutation from rejected invocation.

## Errors

- Exit 0 with extra positional arg accepted.
- Silent failure without usage hint.

## Exit Code

Non-zero.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure for extra args; combined:\n%s", resp.Combined)
	}

	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "foo") {
		t.Fatalf("error should reference unexpected arg; combined:\n%s", resp.Combined)
	}
	hasUsageHint := strings.Contains(lower, "usage") ||
		strings.Contains(lower, "argument") ||
		strings.Contains(lower, "args") ||
		strings.Contains(lower, "paste-bin")
	if !hasUsageHint {
		t.Fatalf("expected usage/args hint; combined:\n%s", resp.Combined)
	}

	assert.Output(t, resp.Combined, `<contains>
paste-bin
</contains>`)
	assertScratchContentExact(t, resp.ScratchAfter.Content, seededUTF8Content)
}
```