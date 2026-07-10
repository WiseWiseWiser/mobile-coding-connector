## Expected

1. `Label` is exactly `Loading…`.
2. Must not be `No wrk projects` (loading is not an empty registry).

## Errors

- Treating loading empty as idle empty registry.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Loading…"
	if resp.Label != want {
		t.Fatalf("Label = %q, want %q (empty+loading is not empty registry)", resp.Label, want)
	}
}
```
