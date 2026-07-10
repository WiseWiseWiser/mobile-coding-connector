## Expected

1. `Label` is exactly `Failed to load projects`.

## Errors

- Shows empty registry wording on hard failure with no rows.
- Leaves Loading… stuck after failure.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Failed to load projects"
	if resp.Label != want {
		t.Fatalf("Label = %q, want %q", resp.Label, want)
	}
}
```
