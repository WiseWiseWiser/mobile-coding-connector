## Expected

1. `ParseErr` is empty.
2. `MonthlyUsage` is `42%`.
3. `CreditsUsed` is `3,000`.
4. `CreditsTotal` is `8,000`.
5. `NextReset` is `09:15 on 15 Sep`.

## Errors

- Parser confused by surrounding noise.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("parse error: %s", resp.ParseErr)
	}
	if resp.MonthlyUsage != "42%" {
		t.Fatalf("MonthlyUsage = %q, want 42%%", resp.MonthlyUsage)
	}
	if resp.CreditsUsed != "3,000" {
		t.Fatalf("CreditsUsed = %q, want 3,000", resp.CreditsUsed)
	}
	if resp.CreditsTotal != "8,000" {
		t.Fatalf("CreditsTotal = %q, want 8,000", resp.CreditsTotal)
	}
	if resp.NextReset != "09:15 on 15 Sep" {
		t.Fatalf("NextReset = %q", resp.NextReset)
	}
}
```