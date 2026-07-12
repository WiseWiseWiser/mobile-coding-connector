## Expected

1. `ParseErr` is empty.
2. `MonthlyUsage` is `68%` (fixture shows 32% left → parser stores used percent `100-left`).
3. `CreditsUsed` is `7698` (commas stripped by `ParseStatusSnapshot`).
4. `CreditsTotal` is `11250`.
5. `NextReset` is `08:00 on 1 Aug` (first `(resets …)` group).
6. `IsBlockingMenu` is false (status screen is not the update modal).

## Errors

- Parse failure or wrong fields.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("ParseStatusSnapshot error: %s", resp.ParseErr)
	}
	// Fixture: "32% left" → usage percent 68%.
	if resp.MonthlyUsage != "68%" {
		t.Fatalf("MonthlyUsage=%q, want 68%% (from 32%% left)", resp.MonthlyUsage)
	}
	if resp.CreditsUsed != "7698" {
		t.Fatalf("CreditsUsed=%q, want 7698 (comma-stripped)", resp.CreditsUsed)
	}
	if resp.CreditsTotal != "11250" {
		t.Fatalf("CreditsTotal=%q, want 11250", resp.CreditsTotal)
	}
	if resp.NextReset != "08:00 on 1 Aug" {
		t.Fatalf("NextReset=%q, want 08:00 on 1 Aug", resp.NextReset)
	}
	if resp.IsBlockingMenu {
		t.Fatalf("status screen must not be IsBlockingMenu")
	}
}
```
