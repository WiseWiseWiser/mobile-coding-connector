## Expected

1. `Label` is exactly `Failed to load projects`.

## Errors

- Generic `Error` or empty string.
- Reuses empty-registry wording.

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
