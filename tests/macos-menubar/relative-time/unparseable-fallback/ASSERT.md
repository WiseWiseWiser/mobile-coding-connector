## Expected

1. `TimeLeft` is empty.

## Errors

- Non-empty output or error instead of silent fallback.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "" {
		t.Fatalf("TimeLeft = %q, want empty", resp.TimeLeft)
	}
}
```