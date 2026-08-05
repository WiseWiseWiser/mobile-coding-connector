## Expected

1. `LoadErr` is non-empty (corrupt JSON must surface as error).
2. `Loaded` is nil.

## Side Effects

- Corrupt file remains under `{Root}/ssh-sessions/` (test artifact).

## Errors

- Expected: Load returns a non-nil error (message may mention json/invalid/parse).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.LoadErr == "" {
		t.Fatal("Load corrupt JSON must return an error; LoadErr is empty")
	}
	if resp.Loaded != nil {
		t.Fatalf("Loaded must be nil on corrupt JSON; got %+v", resp.Loaded)
	}
}
```
