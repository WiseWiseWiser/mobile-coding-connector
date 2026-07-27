## Expected

1. `Line` is exactly
   `Wrote /Users/u/.backup/ai-critic/foo.example.com/machine-backup-20260710-150000.tar.xz (42 MB)`.

## Errors

- MiB label, decimal sizes, missing path, or `Wrote:` with colon only.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Wrote /Users/u/.backup/ai-critic/foo.example.com/machine-backup-20260710-150000.tar.xz (42 MB)"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
