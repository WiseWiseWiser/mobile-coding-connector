## Expected

1. Parsed `Mode` is `sshcmd.ModeCommand` when Parse succeeds.
2. Parsed `RemoteArgv` is `["ls"]` when Parse succeeds.
3. Error text contains
   `no active SSH tunnel; run 'remote-agent ssh --serve' first`
4. `RunnerCalls` is 0; `ServeStartCalls` is 0.
5. `StoreLoadCalls` is at least 1.

## Side Effects

- No runner invocation without Alive session.

## Errors

- Expected tunnel missing error.

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
	if resp.ParseErr == "" {
		if resp.Mode != sshcmd.ModeCommand {
			t.Fatalf("Mode: got %q want %q", resp.Mode, sshcmd.ModeCommand)
		}
		if len(resp.RemoteArgv) != 1 || resp.RemoteArgv[0] != "ls" {
			t.Fatalf("RemoteArgv: got %v want [ls]", resp.RemoteArgv)
		}
	}
	msg := errText(resp)
	if !strings.Contains(msg, noSessionMsg) {
		t.Fatalf("error must contain %q; got %q", noSessionMsg, msg)
	}
	if resp.RunnerCalls != 0 {
		t.Fatalf("SSHRunner must not run without session; calls=%d", resp.RunnerCalls)
	}
	if resp.ServeStartCalls != 0 {
		t.Fatalf("ServeStarter must not Start; calls=%d", resp.ServeStartCalls)
	}
	if resp.StoreLoadCalls < 1 {
		t.Fatalf("SessionStore.Load should be called; calls=%d", resp.StoreLoadCalls)
	}
}
```
