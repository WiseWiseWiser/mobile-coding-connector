## Expected

1. `ManualShowsWindow` is true.

## Side Effects

- None (read-only source inspection).

## Errors

- Backup Now runs with no window open.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ManualShowsWindow {
		t.Fatalf("manual Backup Now does not open progress window (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
