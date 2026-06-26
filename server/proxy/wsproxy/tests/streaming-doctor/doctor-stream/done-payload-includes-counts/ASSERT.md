## Expected

1. Final event is `type: done`.
2. `done.healthy` is present (bool).
3. `done.try_url` equals `Request.TryURL`.
4. `done.checks_total` is a positive integer.
5. `done.checks_failed` is present and `≤ checks_total`.
6. `checks_failed` reflects the number of server progress events with `status: fail`.

## Side Effects

None.

## Errors

- Missing `checks_total` or `checks_failed` on done frame.
- `try_url` does not echo the request parameter.

```go
import (
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.DoneHealthy == nil {
		t.Fatal("done.healthy missing")
	}
	if resp.DoneTryURL != req.TryURL {
		t.Fatalf("done.try_url = %q, want %q", resp.DoneTryURL, req.TryURL)
	}
	if resp.DoneChecksTotal <= 0 {
		t.Fatalf("done.checks_total = %d, want > 0", resp.DoneChecksTotal)
	}
	if resp.DoneChecksFailed < 0 || resp.DoneChecksFailed > resp.DoneChecksTotal {
		t.Fatalf("done.checks_failed = %d, checks_total = %d (invalid)", resp.DoneChecksFailed, resp.DoneChecksTotal)
	}

	failCount := 0
	for _, ev := range serverProgressLayer(resp.Events) {
		if status, _ := ev.Decoded["status"].(string); status == "fail" {
			failCount++
		}
	}
	if resp.DoneChecksFailed != failCount {
		t.Fatalf("done.checks_failed = %d, counted fail progress events = %d", resp.DoneChecksFailed, failCount)
	}
}
```
