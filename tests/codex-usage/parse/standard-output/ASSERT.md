## Expected

1. `ParseErr` is empty.
2. `MonthlyUsage` is `58%`.
3. `CreditsUsed` is `6,519`.
4. `CreditsTotal` is `11,250`.
5. `NextReset` is `08:00 on 1 Aug`.

## Errors

- Parse failure or wrong field values.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("parse error: %s", resp.ParseErr)
	}
	if resp.MonthlyUsage != "58%" {
		t.Fatalf("MonthlyUsage = %q, want 58%%", resp.MonthlyUsage)
	}
	if resp.CreditsUsed != "6,519" {
		t.Fatalf("CreditsUsed = %q, want 6,519", resp.CreditsUsed)
	}
	if resp.CreditsTotal != "11,250" {
		t.Fatalf("CreditsTotal = %q, want 11,250", resp.CreditsTotal)
	}
	if resp.NextReset != "08:00 on 1 Aug" {
		t.Fatalf("NextReset = %q", resp.NextReset)
	}
}
```