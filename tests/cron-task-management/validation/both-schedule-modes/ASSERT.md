## Expected

1. Create fails: `ActionError` non-empty and/or `HTTPStatus` not in 2xx.
2. List does not contain `both-modes`.

## Side Effects

- No persisted invalid definition.

## Errors

- Create succeeds with both schedule modes.

## Exit Code

0 from harness `Run` (failure is product error, not harness crash).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness error: %v", err)
	}
	// Require a real validation response from the cron API (not missing route 404).
	if resp.HTTPStatus == 404 || resp.HTTPStatus == 0 {
		t.Fatalf("cron create API missing or unusable; status=%d body=%s err=%s",
			resp.HTTPStatus, resp.Body, resp.ActionError)
	}
	if resp.HTTPStatus < 400 || resp.HTTPStatus >= 500 {
		t.Fatalf("want 4xx validation status for both schedules; status=%d body=%s err=%s",
			resp.HTTPStatus, resp.Body, resp.ActionError)
	}
	msg := strings.ToLower(resp.Body + " " + resp.ActionError)
	if !strings.Contains(msg, "schedule") && !strings.Contains(msg, "interval") && !strings.Contains(msg, "cron") {
		t.Fatalf("validation error should mention schedule/interval/cron; body=%s err=%s",
			resp.Body, resp.ActionError)
	}
	if _, found := findTaskByName(resp.Tasks, "both-modes"); found {
		t.Fatal("both-modes should not be listed")
	}
}
```

