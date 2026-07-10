## Expected

1. `Line` is exactly `Downloading archive…` (Unicode ellipsis).

## Errors

- Three ASCII dots `...`, different casing, or missing word archive.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Downloading archive…"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
