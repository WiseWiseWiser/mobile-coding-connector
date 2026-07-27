## Expected

1. `CanRun` is `true` (one-shot works while task is disabled).

## Errors

- Gating Backup Now on `enabled` / requiring Enable first.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.CanRun {
		t.Fatal("CanRun = false, want true when disabled but endpoint ready (one-shot)")
	}
}
```
