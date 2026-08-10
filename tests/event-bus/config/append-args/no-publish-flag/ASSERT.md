## Expected

1. No error.
2. ArgsOut contains `--no-event-bus-publish`.
3. Base args preserved as prefix.

## Errors

- Missing disable flag for keep-alive child.

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
	if len(resp.ArgsOut) < 1 || resp.ArgsOut[0] != req.BaseArgs[0] {
		t.Fatalf("base args not preserved: %v", resp.ArgsOut)
	}
	found := false
	for _, a := range resp.ArgsOut {
		if a == "--no-event-bus-publish" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing --no-event-bus-publish in %v", resp.ArgsOut)
	}
}
```
