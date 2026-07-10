## Expected

1. Create succeeds.
2. `len(History) >= 2`.
3. Each run has non-empty `StartedAt` parseable as RFC3339 (UTC).

## Side Effects

- Runs accumulate for the task within 7d window.

## Errors

- Empty history after multiple interval fires.

## Exit Code

0 from `Run`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("create failed: %s", resp.ActionError)
	}
	if len(resp.History) < 2 {
		t.Fatalf("want ≥2 history runs, got %d: %+v", len(resp.History), resp.History)
	}
	for i, r := range resp.History {
		if r.StartedAt == "" {
			t.Fatalf("history[%d].StartedAt empty", i)
		}
		if _, err := parseRFC3339(r.StartedAt); err != nil {
			t.Fatalf("history[%d].StartedAt %q not RFC3339: %v", i, r.StartedAt, err)
		}
	}
}
```
