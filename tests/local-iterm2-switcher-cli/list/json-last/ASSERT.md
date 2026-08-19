## Expected

1. Exit 0.
2. No HTTP leak.
3. Stdout parses as inventory JSON with `sess-a`; no ANSI; trailing `\n`.

## Errors

- Unknown command; human text; daemon HTTP.

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
	if resp.HitHTTP {
		t.Fatalf("list --json must not talk to daemon; out=%q err=%q", resp.Combined, resp.ErrMsg)
	}
	if strings.Contains(resp.Stdout, "\x1b[") {
		t.Fatal("--json must not include ANSI")
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatal("stdout must end with newline")
	}
	var inv localiterm2.Inventory
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &inv); err != nil {
		t.Fatalf("stdout not inventory JSON: %v\n%s", err, resp.Stdout)
	}
	n := 0
	hasA := false
	for _, d := range inv.Desktops {
		n += len(d.Sessions)
		for _, s := range d.Sessions {
			if s.SessionID == "sess-a" {
				hasA = true
			}
		}
	}
	if n < 1 || !hasA {
		t.Fatalf("last inventory missing sess-a sessions=%d", n)
	}
}
```
