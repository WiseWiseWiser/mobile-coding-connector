## Expected

1. No error.
2. `Config.Disabled == true`.

## Errors

- Disabled false when noPublish set.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !resp.Config.Disabled {
		t.Fatal("Config.Disabled false, want true when noPublish")
	}
}
```
