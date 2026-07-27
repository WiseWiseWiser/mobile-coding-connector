## Expected

1. `Title` is exactly `nightly ○ Idle (disabled) · cron 0 1 * * *`.

## Errors

- Wrong glyph, status word, schedule suffix, separators, or spacing.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := `nightly ○ Idle (disabled) · cron 0 1 * * *`
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
