## Expected

1. `ServerName` is exactly `foo.example.com` (host only; no scheme, path, or slash).

## Errors

- Scheme retained, trailing slash, or path segments included.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "foo.example.com"
	if resp.ServerName != want {
		t.Fatalf("ServerName = %q, want %q", resp.ServerName, want)
	}
}
```
