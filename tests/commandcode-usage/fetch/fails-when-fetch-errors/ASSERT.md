## Expected

1. Status is `error`.
2. The error message is the fetcher's message, unchanged.

## Errors

- Wrapping or replacing the provider message.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "error" || resp.Error != "token rejected by provider" {
		t.Fatalf("status = %q, error = %q", resp.Status, resp.Error)
	}
}
```
