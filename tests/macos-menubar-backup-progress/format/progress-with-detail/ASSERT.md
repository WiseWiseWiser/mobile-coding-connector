## Expected

1. `Line` is exactly `[progress] home ok — 12 files`.

## Errors

- Colon instead of em dash, missing spaces, or dropping detail.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "[progress] home ok — 12 files"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
