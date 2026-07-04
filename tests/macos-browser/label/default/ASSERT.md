## Expected

1. `Label` is exactly `Open in Browser`.

## Errors

- Suffix appended for default preference.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "Open in Browser" {
		t.Fatalf("label = %q, want %q", resp.Label, "Open in Browser")
	}
}
```