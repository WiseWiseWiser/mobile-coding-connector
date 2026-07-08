## Expected

1. `Title` is exactly `web ● Running`.

## Errors

- Wrong indicator, spacing, or status label.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Title != "web ● Running" {
		t.Fatalf("title = %q, want %q", resp.Title, "web ● Running")
	}
}
```