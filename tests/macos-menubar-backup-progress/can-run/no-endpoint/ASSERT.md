## Expected

1. `CanRun` is `false`.

## Errors

- Enabling Backup Now with no server endpoint.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanRun {
		t.Fatal("CanRun = true, want false without endpoint")
	}
}
```
