## Expected

1. `Line` is exactly `[done] archive ready`.

## Errors

- Omitting default text, or using raw JSON of the done frame.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "[done] archive ready"
	if resp.Line != want {
		t.Fatalf("Line = %q, want %q", resp.Line, want)
	}
}
```
