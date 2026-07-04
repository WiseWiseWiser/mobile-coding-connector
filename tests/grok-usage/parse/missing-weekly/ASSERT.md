## Expected

1. `ParseErr` is non-empty.
2. `WeeklyLimit` is empty.

## Errors

- Parser returns success without weekly limit.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr == "" {
		t.Fatal("expected parse error for missing weekly limit")
	}
	if resp.WeeklyLimit != "" {
		t.Fatalf("WeeklyLimit = %q, want empty on error", resp.WeeklyLimit)
	}
}
```