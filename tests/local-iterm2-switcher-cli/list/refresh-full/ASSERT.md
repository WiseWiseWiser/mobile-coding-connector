## Expected

1. Exit 0.
2. No HTTP leak.
3. `CaptureCalls >= 1` (full recapture, not Layout-only).
4. JSON includes `sess-b`.

## Errors

- Incremental-only without Capture; missing sess-b; daemon HTTP.

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
		t.Fatalf("--refresh must not talk to daemon; out=%q err=%q", resp.Combined, resp.ErrMsg)
	}
	if resp.CaptureCalls < 1 {
		t.Fatalf("CaptureCalls=%d want ≥ 1 (full recapture)", resp.CaptureCalls)
	}
	var inv localiterm2.Inventory
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &inv); err != nil {
		t.Fatalf("stdout not inventory JSON: %v\n%s", err, resp.Stdout)
	}
	hasB := false
	for _, d := range inv.Desktops {
		for _, s := range d.Sessions {
			if s.SessionID == "sess-b" {
				hasB = true
			}
		}
	}
	if !hasB {
		t.Fatalf("refresh must recapture sess-b; stdout=%s", resp.Stdout)
	}
}
```
