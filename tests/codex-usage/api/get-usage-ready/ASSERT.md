## Expected

1. `APIStatusCode` is `200`.
2. `APIParsed.Status` is `ready`.
3. `APIParsed.MonthlyUsage` is `58%`.
4. `APIParsed.CreditsUsed` is `6,519`.
5. `APIParsed.CreditsTotal` is `11,250`.
6. `APIParsed.NextReset` is `08:00 on 1 Aug`.
7. `APIParsed.UpdatedAt` is non-empty.

## Errors

- API stuck in loading or returns error status.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.APIStatusCode != 200 {
		t.Fatalf("status code = %d body=%s", resp.APIStatusCode, resp.APIBody)
	}
	if resp.APIParsed == nil {
		t.Fatalf("could not parse API body: %s", resp.APIBody)
	}
	if resp.APIParsed.Status != "ready" {
		t.Fatalf("json status = %q, want ready; body=%s", resp.APIParsed.Status, resp.APIBody)
	}
	if resp.APIParsed.MonthlyUsage != "58%" {
		t.Fatalf("monthly_usage = %q", resp.APIParsed.MonthlyUsage)
	}
	if resp.APIParsed.CreditsUsed != "6,519" {
		t.Fatalf("credits_used = %q", resp.APIParsed.CreditsUsed)
	}
	if resp.APIParsed.CreditsTotal != "11,250" {
		t.Fatalf("credits_total = %q", resp.APIParsed.CreditsTotal)
	}
	if resp.APIParsed.NextReset != "08:00 on 1 Aug" {
		t.Fatalf("next_reset = %q", resp.APIParsed.NextReset)
	}
	if resp.APIParsed.UpdatedAt == "" {
		t.Fatal("updated_at missing in API response")
	}
}
```