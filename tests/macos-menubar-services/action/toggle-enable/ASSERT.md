## Expected

1. `ShowEnable` is `true` (menu offers **Enable**, not Disable).

## Errors

- Disable action shown for a disabled service.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.ShowEnable {
		t.Fatal("ShowEnable = false, want true for disabled service")
	}
}
```