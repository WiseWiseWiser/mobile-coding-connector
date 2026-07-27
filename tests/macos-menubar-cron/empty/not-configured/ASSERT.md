## Expected

1. `EmptyLabel` is exactly `Not configured`.

## Errors

- Long status-line copy used inside the Cron menu, or empty string.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Not configured"
	if resp.EmptyLabel != want {
		t.Fatalf("not-configured label = %q, want %q", resp.EmptyLabel, want)
	}
}
```
