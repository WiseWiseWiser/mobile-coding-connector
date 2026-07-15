## Expected Output

```
<contains>
unauthorized
</contains>
```

## Expected

1. Non-zero exit.
2. Combined output indicates unauthorized/auth failure.
3. Seeded scratch content unchanged on server.

## Side Effects

No scratch mutation from failed auth.

## Errors

- Exit 0 with bad token.
- Silent failure without auth messaging.

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
		t.Fatalf("expected auth failure; combined:\n%s", resp.Combined)
	}

	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "unauthorized") && !strings.Contains(lower, "auth") {
		t.Fatalf("expected unauthorized/auth failure; combined:\n%s", resp.Combined)
	}

	assert.Output(t, resp.Combined, `<contains>
unauthorized
</contains>`)
	assertScratchContentExact(t, resp.ScratchAfter.Content, seededUTF8Content)
}
```