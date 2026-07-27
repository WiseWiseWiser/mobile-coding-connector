## Expected

1. `StatusTitle` is exactly `Status: On · Running`.

## Errors

- Progress percent, spinner text, or missing On prefix.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Status: On · Running"
	if resp.StatusTitle != want {
		t.Fatalf("StatusTitle = %q, want %q", resp.StatusTitle, want)
	}
}
```
