## Expected

1. `ShouldRun` is `false` (no overlap).

## Errors

- Parallel/overlapping backup runs for the same server.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ShouldRun {
		t.Fatal("ShouldRun = true, want false while already running")
	}
}
```
