## Expected

1. `EmptyLabel` is exactly `No terminal sessions`.

## Errors

- Missing or alternate placeholder (`No sessions`, `Empty`, etc.).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.EmptyLabel != "No terminal sessions" {
		t.Fatalf("empty label = %q, want %q", resp.EmptyLabel, "No terminal sessions")
	}
}
```
