## Expected

1. HTTP 200.
2. Body JSON: `"sessions": []` (empty array, not null).
3. `resp.Sessions` length 0.

## Errors

- Non-200 status.
- Non-empty sessions.

## Exit Code

0 (harness maps HTTP 200 → ExitCode 0).

```go
import (
	"encoding/json"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.HTTPStatus != 200 {
		t.Fatalf("HTTP %d body:\n%s", resp.HTTPStatus, resp.Body)
	}
	if len(resp.Sessions) != 0 {
		t.Fatalf("want empty sessions, got %+v", resp.Sessions)
	}
	var payload struct {
		Sessions []SessionItem `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err != nil {
		t.Fatalf("JSON: %v body=%s", err, resp.Body)
	}
	if payload.Sessions == nil {
		t.Fatalf("sessions must be [] not null; body=%s", resp.Body)
	}
	if len(payload.Sessions) != 0 {
		t.Fatalf("want [], got %+v", payload.Sessions)
	}
}
```
