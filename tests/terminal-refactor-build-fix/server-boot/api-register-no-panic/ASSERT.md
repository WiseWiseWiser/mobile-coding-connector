# Expected

`terminal.RegisterAPI` must register all terminal routes on a fresh
`http.ServeMux` **without panicking**. The server must be able to boot.

- No panic: `resp.RegisterPaniced == false`.
- No panic text: `resp.RegisterPanic == ""`.

The fixed behavior: the adapter must not double-register `/api/terminal`.
`ptywrap.RegisterAPIWithManager` already owns `/api/terminal`; the SSH wrapper
must layer on top without re-registering the same pattern (e.g. by registering
the wrapper before delegating, or by having ptywrap expose a hook for the SSH
branch), so Go 1.22+ `ServeMux` does not panic on a conflicting duplicate.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.RegisterPaniced {
		t.Fatalf("terminal.RegisterAPI must not panic on boot, but panicked: %s", resp.RegisterPanic)
	}
	if resp.RegisterPanic != "" {
		t.Fatalf("expected empty panic text, got %q", resp.RegisterPanic)
	}
}
```
