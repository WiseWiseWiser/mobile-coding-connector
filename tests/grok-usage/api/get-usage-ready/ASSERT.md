## Expected

1. `APIStatusCode` is `200`.
2. `APIParsed.Status` is `ready`.
3. `APIParsed.WeeklyLimit` is `6%`.
4. `APIParsed.NextReset` is `July 9, 16:55 PT`.
5. `APIParsed.UpdatedAt` is non-empty.

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
	if resp.APIParsed.WeeklyLimit != "6%" {
		t.Fatalf("weekly_limit = %q", resp.APIParsed.WeeklyLimit)
	}
	if resp.APIParsed.NextReset != "July 9, 16:55 PT" {
		t.Fatalf("next_reset = %q", resp.APIParsed.NextReset)
	}
	if resp.APIParsed.UpdatedAt == "" {
		t.Fatal("updated_at missing in API response")
	}
}
```