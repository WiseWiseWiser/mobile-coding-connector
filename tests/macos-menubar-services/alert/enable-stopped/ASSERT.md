## Expected

1. `AlertMessage` is exactly `The server won't start immediately until daemon checks at next time`.

## Errors

- Alert copy diverges from `server/services` `msgEnableStopped`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AlertMessage != msgEnableStopped {
		t.Fatalf("alert = %q, want %q", resp.AlertMessage, msgEnableStopped)
	}
}
```