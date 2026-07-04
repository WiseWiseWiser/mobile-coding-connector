## Expected

1. `Label` is exactly `Grok timeout waiting for usage`.

## Errors

- Truncation applied to short error message.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok timeout waiting for usage"
	if resp.Label != want {
		t.Fatalf("label = %q, want %q", resp.Label, want)
	}
}
```