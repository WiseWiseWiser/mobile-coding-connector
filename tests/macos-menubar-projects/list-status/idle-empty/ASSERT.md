## Expected

1. `Label` is exactly `No wrk projects`.

## Errors

- Shows Loading… after load completed.
- Shows failed label without an error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "No wrk projects"
	if resp.Label != want {
		t.Fatalf("Label = %q, want %q", resp.Label, want)
	}
}
```
