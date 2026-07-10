## Expected

1. `Title` is exactly `feat-login ● Clean`.

## Errors

- Missing basename or wrong clean marker.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "feat-login ● Clean"
	if resp.Title != want {
		t.Fatalf("title = %q, want %q", resp.Title, want)
	}
}
```
