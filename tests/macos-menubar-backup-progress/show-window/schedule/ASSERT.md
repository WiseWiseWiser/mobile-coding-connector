## Expected

1. `ShowWindow` is `false`.

## Errors

- Popping a progress window on every hourly tick.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ShowWindow {
		t.Fatal("ShowWindow = true, want false for schedule-triggered runs")
	}
}
```
