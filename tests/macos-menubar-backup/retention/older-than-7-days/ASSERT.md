## Expected

1. Keep set empty.
2. Delete contains `old-8d.tar.xz`.

## Errors

- Retaining archives beyond the 7-day history window.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.KeepPaths) != 0 {
		t.Fatalf("KeepPaths = %v, want empty", resp.KeepPaths)
	}
	if !pathSetEqual(resp.DeletePaths, []string{"old-8d.tar.xz"}) {
		t.Fatalf("DeletePaths = %v, want [old-8d.tar.xz]", resp.DeletePaths)
	}
}
```
