## Expected

1. `SortedPaths` is exactly `new.tar.xz`, `mid.tar.xz`, `old.tar.xz` (newest first).

## Errors

- Oldest-first, name sort, or unstable order.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"new.tar.xz", "mid.tar.xz", "old.tar.xz"}
	if len(resp.SortedPaths) != len(want) {
		t.Fatalf("SortedPaths len=%d %v, want %v", len(resp.SortedPaths), resp.SortedPaths, want)
	}
	for i := range want {
		if resp.SortedPaths[i] != want[i] {
			t.Fatalf("SortedPaths = %v, want %v", resp.SortedPaths, want)
		}
	}
}
```
