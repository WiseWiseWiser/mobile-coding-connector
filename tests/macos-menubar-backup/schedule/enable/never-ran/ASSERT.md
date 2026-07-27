## Expected

1. `ShouldRun` is `true` (run immediately on enable).

## Errors

- Deferring first run until next hour tick.

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
		t.Fatal("ShouldRun = false, want true when never ran")
	}
}
```
