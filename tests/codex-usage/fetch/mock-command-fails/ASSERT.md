## Expected

1. `ServiceStatus` is `error`.
2. `ServiceError` is non-empty.
3. `MonthlyUsage`, `CreditsUsed`, `CreditsTotal`, and `NextReset` are empty.

## Errors

- Service reports ready on failing script.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServiceStatus != "error" {
		t.Fatalf("status = %q, want error", resp.ServiceStatus)
	}
	if resp.ServiceError == "" {
		t.Fatal("error message empty")
	}
	if resp.MonthlyUsage != "" || resp.CreditsUsed != "" || resp.CreditsTotal != "" || resp.NextReset != "" {
		t.Fatalf("usage fields should be empty on error: monthly=%q credits_used=%q credits_total=%q reset=%q",
			resp.MonthlyUsage, resp.CreditsUsed, resp.CreditsTotal, resp.NextReset)
	}
}
```