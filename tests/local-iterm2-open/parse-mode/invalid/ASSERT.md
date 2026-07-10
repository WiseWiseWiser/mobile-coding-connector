## Expected

1. `ParseErr` is non-empty.

## Errors

- Accepting unknown modes as smart/reuse.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr == "" {
		t.Fatalf("want parse error for ModeInput=%q, got mode=%v", req.ModeInput, resp.ParsedMode)
	}
}
```
