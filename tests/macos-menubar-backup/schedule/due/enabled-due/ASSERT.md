## Expected

1. `ShouldRun` is `true`.

## Errors

- Skipping a due run while idle.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShouldRun {
		t.Fatal("ShouldRun = false, want true when enabled, due, not running")
	}
}
```
