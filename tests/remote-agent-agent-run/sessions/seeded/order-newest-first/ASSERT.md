## Expected

1. Exit code 0.
2. JSON `sessions` array order is newest-first by `updated_at`:
   `sess-new`, then `sess-mid`, then `sess-old`.

## Errors

- Wrong order.
- Non-zero exit.

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
	var payload struct {
		Sessions []SessionItem `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("JSON decode: %v; stdout:\n%s", err, out)
	}
	if len(payload.Sessions) != 3 {
		t.Fatalf("want 3 sessions, got %d", len(payload.Sessions))
	}
	want := []string{"sess-new", "sess-mid", "sess-old"}
	for i, id := range want {
		if payload.Sessions[i].SessionID != id {
			got := make([]string, len(payload.Sessions))
			for j, s := range payload.Sessions {
				got[j] = s.SessionID
			}
			t.Fatalf("order want %v, got %v", want, got)
		}
	}
}
```
