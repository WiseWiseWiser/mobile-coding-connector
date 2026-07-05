## Expected

1. `Label` is exactly `Grok err` (short fixed label for any error length).

## Errors

- Long error truncated in menu bar instead of fixed `Grok err`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Grok err"
	if resp.Label != want {
		t.Fatalf("label = %q, want %q", resp.Label, want)
	}
}
```