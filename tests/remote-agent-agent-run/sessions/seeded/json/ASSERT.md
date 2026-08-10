## Expected

1. Exit code 0.
2. Stdout is a JSON object with `sessions` array of length 3.
3. Each item includes `session_id`, `runner`, `status` matching seeds.
4. No ANSI escape sequences.

## Errors

- Non-JSON output.
- Missing fields or wrong count.

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
		t.Fatalf("JSON decode: %v; stdout:\n%s", err, out)
	}
	if len(payload.Sessions) != 3 {
		t.Fatalf("want 3 sessions, got %d: %+v", len(payload.Sessions), payload.Sessions)
	}
	byID := map[string]SessionItem{}
	for _, s := range payload.Sessions {
		byID[s.SessionID] = s
	}
	want := map[string]struct{ Runner, Status string }{
		"sess-new": {"opencode", "idle"},
		"sess-mid": {"grok", "running"},
		"sess-old": {"codex", "finished"},
	}
	for id, w := range want {
		got, ok := byID[id]
		if !ok {
			t.Fatalf("missing session %s in %+v", id, payload.Sessions)
		}
		if got.Runner != w.Runner {
			t.Fatalf("%s runner=%q want %q", id, got.Runner, w.Runner)
		}
		if got.Status != w.Status {
			t.Fatalf("%s status=%q want %q", id, got.Status, w.Status)
		}
	}
}
```
