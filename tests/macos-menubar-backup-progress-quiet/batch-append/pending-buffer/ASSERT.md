## Expected

1. `HasPendingBuffer` is true (pending/buffer + flush wiring).

## Side Effects

- None (read-only source inspection).

## Errors

- Only immediate `appendOnMain` writing the text view with no pending buffer.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.HasPendingBuffer {
		t.Fatalf("expected pending/buffer + flush symbols in BackupProgressWindow (source: %s)", resp.ProgressWindowSource)
	}
}
```
