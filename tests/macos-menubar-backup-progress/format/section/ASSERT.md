## Expected

1. `Line` is exactly `[section] Collecting files`.

## Errors

- Missing brackets, wrong type tag, or Title: prefix from CLI streamcmd.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "[section] Collecting files"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
