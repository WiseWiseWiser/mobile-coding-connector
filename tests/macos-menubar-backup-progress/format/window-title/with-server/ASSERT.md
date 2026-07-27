## Expected

1. `Title` / `Line` is exactly `Backup: foo.example.com`.

## Errors

- `Logs:` prefix, missing space after colon, or raw URL.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Backup: foo.example.com"
	if resp.Title != want && resp.Line != want {
		t.Fatalf("Title/Line = %q / %q, want %q", resp.Title, resp.Line, want)
	}
}
```
