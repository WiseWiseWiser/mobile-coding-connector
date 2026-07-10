## Expected

1. `Label` is exactly `No wrk projects`.

## Errors

- Shows Loading… after load completed.
- Shows failed label without an error.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "No wrk projects"
	if resp.Label != want {
		t.Fatalf("Label = %q, want %q", resp.Label, want)
	}
}
```
