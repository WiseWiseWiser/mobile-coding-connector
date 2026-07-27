## Expected

1. `AlertMessage` is exactly `Task updated`.

## Errors

- Empty alert body; alternate fallback wording.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Task updated"
	if resp.AlertMessage != want {
		t.Fatalf("alert = %q, want %q", resp.AlertMessage, want)
	}
}
```
