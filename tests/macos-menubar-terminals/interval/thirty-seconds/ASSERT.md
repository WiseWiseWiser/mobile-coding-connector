## Expected

1. `RefreshIntervalSec` is exactly `30`.

## Errors

- Different poll period without updating this sealed contract.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.RefreshIntervalSec != 30 {
		t.Fatalf("refresh interval = %ds, want 30s", resp.RefreshIntervalSec)
	}
}
```
