## Expected

1. Exit 0.
2. Output includes session id `sess-st` (or inject Session label).
3. Mentions multi-layer concepts: at least two of process / terminal / runner /
   resume / status / workspace (human or JSON keys).
4. If JSON: parseable object preferred.

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
	out := resp.Stdout
	if !strings.Contains(out, "sess-st") && !strings.Contains(out, "grok/sess-st") {
		t.Fatalf("expected session id in status output; stdout:\n%s", out)
	}
	lower := strings.ToLower(out)
	layers := 0
	for _, k := range []string{"process", "terminal", "runner", "resume", "workspace", "status"} {
		if strings.Contains(lower, k) {
			layers++
		}
	}
	if layers < 2 {
		t.Fatalf("expected multi-layer status fields; stdout:\n%s", out)
	}
	// Soft JSON check when --json used.
	var raw map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &raw); err == nil {
		if len(raw) == 0 {
			t.Fatalf("empty JSON status object")
		}
	}
}
```
