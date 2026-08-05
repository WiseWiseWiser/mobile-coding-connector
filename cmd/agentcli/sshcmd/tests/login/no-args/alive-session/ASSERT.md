## Expected

1. `ParseErr` and `RunErr` are empty.
2. `Mode` is `sshcmd.ModeLogin`.
3. Parsed `RemoteArgv` is empty; `Dest` is empty.
4. `RunnerCalls` is 1; `RunnerRemoteArgv` is empty (len 0).
5. `RunnerSession` is non-nil and `Alive`.
6. `ServeStartCalls` is 0.
7. `StoreLoadCalls` is at least 1.

## Side Effects

- Exactly one SSHRunner.Run with empty remote command list.

## Errors

None expected.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("Parse error: %s", resp.ParseErr)
	}
	if resp.RunErr != "" {
		t.Fatalf("sshcmd.Run error: %s", resp.RunErr)
	}
	if resp.Mode != sshcmd.ModeLogin {
		t.Fatalf("Mode: got %q want %q", resp.Mode, sshcmd.ModeLogin)
	}
	if len(resp.RemoteArgv) != 0 {
		t.Fatalf("parsed RemoteArgv should be empty; got %v", resp.RemoteArgv)
	}
	if resp.Dest != "" {
		t.Fatalf("Dest should be empty for no-args login; got %q", resp.Dest)
	}
	if resp.RunnerCalls != 1 {
		t.Fatalf("SSHRunner.Run calls: got %d want 1", resp.RunnerCalls)
	}
	if len(resp.RunnerRemoteArgv) != 0 {
		t.Fatalf("Runner remote argv want empty; got %v", resp.RunnerRemoteArgv)
	}
	if resp.RunnerSession == nil || !resp.RunnerSession.Alive {
		t.Fatalf("Runner should receive Alive session; got %#v", resp.RunnerSession)
	}
	if resp.ServeStartCalls != 0 {
		t.Fatalf("ServeStarter must not Start in login; calls=%d", resp.ServeStartCalls)
	}
	if resp.StoreLoadCalls < 1 {
		t.Fatalf("SessionStore.Load should be called; calls=%d", resp.StoreLoadCalls)
	}
}
```
