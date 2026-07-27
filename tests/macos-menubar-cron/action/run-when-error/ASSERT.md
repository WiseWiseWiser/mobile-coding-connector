## Expected

1. `CanRun` is `true`.

## Errors

- Run Now enabled/disabled incorrectly for status `error`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanRun != true {
		t.Fatalf("CanRun = %v, want %v for status %q", resp.CanRun, true, req.Status)
	}
}
```
