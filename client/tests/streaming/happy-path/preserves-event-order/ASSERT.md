## Expected

1. `StreamErr` is empty.
2. `CallOrder` equals `section:, progress:p1, meta:, progress:p2, done:` (type:id suffix).
3. `len(Events)` is 4 (excluding done from callback list if Stream strips terminal — otherwise 5).

## Side Effects

None.

## Errors

- Events arrive out of wire order.
- Missing section or meta frames.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.StreamErr != "" {
		t.Fatalf("Stream returned error: %s", resp.StreamErr)
	}
	want := []string{"section:", "progress:p1", "meta:", "progress:p2"}
	if len(resp.CallOrder) < len(want) {
		t.Fatalf("CallOrder = %v, want at least %v", resp.CallOrder, want)
	}
	for i, w := range want {
		if resp.CallOrder[i] != w {
			t.Fatalf("CallOrder[%d] = %q, want %q (full: %v)", i, resp.CallOrder[i], w, resp.CallOrder)
		}
	}
}
```
