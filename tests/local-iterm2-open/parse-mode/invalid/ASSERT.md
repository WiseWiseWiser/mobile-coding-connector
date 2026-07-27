## Expected

1. `ParseErr` is non-empty.

## Errors

- Accepting unknown modes as smart/reuse.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr == "" {
		t.Fatalf("want parse error for ModeInput=%q, got mode=%v", req.ModeInput, resp.ParsedMode)
	}
}
```
