## Expected

1. `Label` is exactly `Grok 6%`.

## Errors

- Rotating index 0 did not select grok slot.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Label != "Grok 6%" {
		t.Fatalf("label = %q, want %q", resp.Label, "Grok 6%")
	}
}
```