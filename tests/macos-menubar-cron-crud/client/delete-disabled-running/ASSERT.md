## Expected

1. `DeleteDisabledWhenRunning` is true (Delete uses `.disabled` and/or
   `canDeleteCronTask` / `CanDeleteCronTask`).

## Side Effects

- None (read-only source inspection).

## Errors

- Delete always enabled; no status gate.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.DeleteDisabledWhenRunning {
		t.Fatalf("Delete not gated when running (sources: %v)", resp.SwiftSourcesChecked)
	}
}
```
