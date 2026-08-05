## Expected

1. Parsed `Mode` is `sshcmd.ModeLogin` when Parse succeeds (before gate).
2. `RunErr` (or combined errText) contains exact contract:
   `no active SSH tunnel; run 'remote-agent ssh --serve' first`
3. Error also implies `--serve` (covered by exact string).
4. `RunnerCalls` is 0.
5. `ServeStartCalls` is 0.
6. `StoreLoadCalls` is at least 1 (gate consulted the store).

## Side Effects

- No SSHRunner invocation.

## Errors

- Expected: no active SSH tunnel message.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/doctest/session"
)

const noSessionMsg = "no active SSH tunnel; run 'remote-agent ssh --serve' first"

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	// Parse should still classify login when possible.
	if resp.ParseErr == "" && resp.Mode != sshcmd.ModeLogin {
		t.Fatalf("Mode: got %q want %q", resp.Mode, sshcmd.ModeLogin)
	}
	msg := errText(resp)
	if !strings.Contains(msg, noSessionMsg) {
		t.Fatalf("error must contain %q; got %q", noSessionMsg, msg)
	}
	if !strings.Contains(msg, "--serve") {
		t.Fatalf("error must mention --serve; got %q", msg)
	}
	if resp.RunnerCalls != 0 {
		t.Fatalf("SSHRunner must not run without session; calls=%d", resp.RunnerCalls)
	}
	if resp.ServeStartCalls != 0 {
		t.Fatalf("ServeStarter must not Start in login; calls=%d", resp.ServeStartCalls)
	}
	if resp.StoreLoadCalls < 1 {
		t.Fatalf("SessionStore.Load should be called; calls=%d", resp.StoreLoadCalls)
	}
}
```
