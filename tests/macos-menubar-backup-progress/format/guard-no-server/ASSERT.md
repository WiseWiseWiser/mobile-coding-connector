## Expected

1. `Line` is exactly `ERROR: no server selected`.

## Errors

- Silent return, or `ERROR: empty server` / different copy.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "ERROR: no server selected"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
