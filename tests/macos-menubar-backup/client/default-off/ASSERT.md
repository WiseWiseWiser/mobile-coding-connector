## Expected

1. `DefaultEnabledFalse` is true (no launch-time force enable; default off).

## Side Effects

- None (read-only source inspection).

## Errors

- Enabling backup automatically in `onAppear` / init without user action.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.DefaultEnabledFalse {
		t.Fatalf("backup appears auto-enabled on launch (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
