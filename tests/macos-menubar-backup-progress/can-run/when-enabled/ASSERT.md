## Expected

1. `CanRun` is `true`.

## Errors

- Disabling Backup Now while the task is on (should stay available).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.CanRun {
		t.Fatal("CanRun = false, want true when enabled and ready")
	}
}
```
