## Expected

1. Exit 0 (or RED until --tty is accepted with --json).
2. Stdout is inventory JSON with sess-a.
3. No box-drawing `┌`, no ANSI escapes.

```go
import (
	"encoding/json"
	"strings"
	"testing"
	"github.com/xhd2015/ai-critic/server/localiterm2"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	if strings.Contains(resp.Stdout, "\x1b[") {
		t.Fatal("--json must not include ANSI (even with --tty)")
	}
	if strings.Contains(resp.Stdout, "┌") || strings.Contains(resp.Stdout, "│") {
		t.Fatal("--json must stay batch (no split box drawing)")
	}
	var inv localiterm2.Inventory
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &inv); err != nil {
		t.Fatalf("stdout not inventory JSON: %v\n%s", err, resp.Stdout)
	}
	hasA := false
	for _, d := range inv.Desktops {
		for _, s := range d.Sessions {
			if s.SessionID == "sess-a" {
				hasA = true
			}
		}
	}
	if !hasA {
		t.Fatal("JSON missing sess-a")
	}
}
```
