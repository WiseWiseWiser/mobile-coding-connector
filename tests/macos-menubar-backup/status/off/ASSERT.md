## Expected

1. `StatusTitle` is exactly `Status: Off`.

## Errors

- Showing On/next while disabled, or different casing/punctuation.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Status: Off"
	if resp.StatusTitle != want {
		t.Fatalf("StatusTitle = %q, want %q", resp.StatusTitle, want)
	}
}
```
