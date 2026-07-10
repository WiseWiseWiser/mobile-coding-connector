## Expected

1. `Line` is exactly `[progress] home ok`.

## Errors

- Including empty detail suffix, wrong tag, or comma separators.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "[progress] home ok"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
