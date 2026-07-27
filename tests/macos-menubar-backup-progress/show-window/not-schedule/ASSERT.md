## Expected

1. `ShowWindow` is `true`.

## Errors

- Silent Backup Now (no progress UI for interactive runs).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShowWindow {
		t.Fatal("ShowWindow = false, want true for non-schedule runs")
	}
}
```
