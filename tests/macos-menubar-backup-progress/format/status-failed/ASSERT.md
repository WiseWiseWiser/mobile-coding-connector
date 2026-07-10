## Expected

1. `Line` is exactly `Status: Failed`.

## Errors

- `Status: Error`, menu-style `Status: On · Error`, or missing colon.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Status: Failed"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
