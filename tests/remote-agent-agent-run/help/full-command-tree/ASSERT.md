## Expected

1. Exit 0.
2. Stdout lists remote-capable commands: sessions, attach, status, resume, send,
   msg, snapshot, watch, kill, run (each name appears at least once).
3. Soft: may mention local-only / focus / web / assets / pty with a note.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; %s", resp.ExitCode, resp.Combined)
	}
	out := strings.ToLower(resp.Stdout)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected non-empty top-level help")
	}
	required := []string{"sessions", "attach", "status", "resume", "send", "msg", "snapshot", "watch", "kill", "run"}
	var missing []string
	for _, c := range required {
		if !strings.Contains(out, c) {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("agent-run --help missing commands %v; stdout:\n%s", missing, resp.Stdout)
	}
}
```
