## Expected

1. `AlertMessage` is exactly `The server won't stop immediately unless you manually stop it`.

## Errors

- Alert copy diverges from `server/services` `msgDisableRunning`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AlertMessage != msgDisableRunning {
		t.Fatalf("alert = %q, want %q", resp.AlertMessage, msgDisableRunning)
	}
}
```