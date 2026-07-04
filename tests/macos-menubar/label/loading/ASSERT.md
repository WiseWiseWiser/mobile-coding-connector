## Expected

1. `Label` is exactly `Grok ...`.

## Errors

- Shows limit or error while still loading.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "Grok ..." {
		t.Fatalf("label = %q, want %q", resp.Label, "Grok ...")
	}
}
```