## Expected

1. `Title` / `Line` is exactly `Backup: (no server)`.

## Errors

- Empty title, `Backup: `, or different placeholder wording.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Backup: (no server)"
	if resp.Title != want && resp.Line != want {
		t.Fatalf("Title/Line = %q / %q, want %q", resp.Title, resp.Line, want)
	}
}
```
