## Expected

1. `ParseErr` is non-empty.
2. `MonthlyUsage` is empty.

## Errors

- Parser returns success without monthly usage.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr == "" {
		t.Fatal("expected parse error for missing monthly usage")
	}
	if resp.MonthlyUsage != "" {
		t.Fatalf("MonthlyUsage = %q, want empty on error", resp.MonthlyUsage)
	}
}
```