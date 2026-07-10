## Expected

1. Create fails (`ActionError` or non-2xx).
2. List does not contain `no-schedule`.

## Side Effects

- None persisted.

## Errors

- Silent success without a schedule.

## Exit Code

0 from harness.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness error: %v", err)
	}
	if resp.HTTPStatus == 404 || resp.HTTPStatus == 0 {
		t.Fatalf("cron create API missing or unusable; status=%d body=%s err=%s",
			resp.HTTPStatus, resp.Body, resp.ActionError)
	}
	if resp.HTTPStatus < 400 || resp.HTTPStatus >= 500 {
		t.Fatalf("want 4xx validation status for missing schedule; status=%d body=%s err=%s",
			resp.HTTPStatus, resp.Body, resp.ActionError)
	}
	msg := strings.ToLower(resp.Body + " " + resp.ActionError)
	if !strings.Contains(msg, "schedule") && !strings.Contains(msg, "interval") && !strings.Contains(msg, "cron") {
		t.Fatalf("validation error should mention schedule/interval/cron; body=%s err=%s",
			resp.Body, resp.ActionError)
	}
	if _, found := findTaskByName(resp.Tasks, "no-schedule"); found {
		t.Fatal("no-schedule should not be listed")
	}
}
```

