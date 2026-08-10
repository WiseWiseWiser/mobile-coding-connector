## Expected

1. Exit code 0.
2. Stdout is JSON object with `"sessions": []` (empty array).
3. No ANSI escape sequences.

## Errors

- Non-zero exit.
- Non-JSON or non-empty sessions array.

## Exit Code

0.

```go
import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := strings.TrimSpace(resp.Stdout)
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("--json must not emit ANSI; stdout:\n%q", out)
	}
	var payload struct {
		Sessions []SessionItem `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("expected JSON object; err=%v stdout:\n%s", err, out)
	}
	if payload.Sessions == nil {
		// null is unacceptable; prefer []
		t.Fatalf("sessions must be [] not null; stdout:\n%s", out)
	}
	if len(payload.Sessions) != 0 {
		t.Fatalf("want 0 sessions, got %d: %+v", len(payload.Sessions), payload.Sessions)
	}
	if resp.Sessions != nil && len(resp.Sessions) != 0 {
		t.Fatalf("parsed Sessions len=%d, want 0", len(resp.Sessions))
	}
}
```
