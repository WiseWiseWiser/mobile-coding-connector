## Expected

1. `ParseErr` is non-empty.
2. `NextReset` is empty.

## Errors

- Parser returns success without next reset.

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
		t.Fatal("expected parse error for missing next reset")
	}
	if resp.NextReset != "" {
		t.Fatalf("NextReset = %q, want empty on error", resp.NextReset)
	}
}
```