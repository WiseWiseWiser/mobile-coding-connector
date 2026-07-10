## Expected

1. `Line` is exactly `ERROR: stream failed`.

## Errors

- Lowercase `error:`, missing colon/space, or `[error]` tag only.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "ERROR: stream failed"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
