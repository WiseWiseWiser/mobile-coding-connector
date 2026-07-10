## Expected

1. `StatusChildren` is exactly `["Enable", "Disable"]` (only two items, this order).

## Errors

- Extra items (Backup Now, Reveal), or only one of Enable/Disable.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Enable", "Disable"}
	if len(resp.StatusChildren) != len(want) {
		t.Fatalf("StatusChildren = %v, want %v", resp.StatusChildren, want)
	}
	for i := range want {
		if resp.StatusChildren[i] != want[i] {
			t.Fatalf("StatusChildren = %v, want %v", resp.StatusChildren, want)
		}
	}
}
```
