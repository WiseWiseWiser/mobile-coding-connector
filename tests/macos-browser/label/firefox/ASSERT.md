## Expected

1. `Label` is exactly `Open in Browser(Firefox)`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Open in Browser(Firefox)"
	if resp.Label != want {
		t.Fatalf("label = %q, want %q", resp.Label, want)
	}
}
```