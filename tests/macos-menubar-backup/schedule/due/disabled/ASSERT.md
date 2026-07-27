## Expected

1. `ShouldRun` is `false`.

## Errors

- Running backups while the task is off.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ShouldRun {
		t.Fatal("ShouldRun = true, want false when disabled")
	}
}
```
