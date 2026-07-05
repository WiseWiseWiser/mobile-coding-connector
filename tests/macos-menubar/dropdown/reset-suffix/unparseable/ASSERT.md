## Expected

1. `ResetSuffix` is empty.

## Errors

- Non-empty suffix or stray comma.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetSuffix != "" {
		t.Fatalf("ResetSuffix = %q, want empty", resp.ResetSuffix)
	}
}
```