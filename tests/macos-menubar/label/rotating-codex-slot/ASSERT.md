## Expected

1. `Label` is exactly `Codex 58%`.

## Errors

- Rotating index 1 did not select codex slot.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "Codex 58%" {
		t.Fatalf("label = %q, want %q", resp.Label, "Codex 58%")
	}
}
```