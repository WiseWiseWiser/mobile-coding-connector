## Expected

1. `ParseErr` and `RunErr` are empty.
2. `Mode` is `sshcmd.ModeLogin`.
3. Parsed `Dest` is `agent@ra` (stripped destination recorded).
4. Parsed `RemoteArgv` is empty.
5. `RunnerCalls` is 1; `RunnerRemoteArgv` is empty (dest not passed as remote cmd).
6. `ServeStartCalls` is 0.

## Side Effects

- Dest consumed by parser; runner sees login-only empty argv.

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
	if resp.Dest != "agent@ra" {
		t.Fatalf("Dest: got %q want %q", resp.Dest, "agent@ra")
	}
	if len(resp.RemoteArgv) != 0 {
		t.Fatalf("RemoteArgv should be empty after dest strip; got %v", resp.RemoteArgv)
	}
	if resp.RunnerCalls != 1 {
		t.Fatalf("SSHRunner.Run calls: got %d want 1", resp.RunnerCalls)
	}
	if len(resp.RunnerRemoteArgv) != 0 {
		t.Fatalf("Runner remote argv must not include dest; got %v", resp.RunnerRemoteArgv)
	}
	if resp.ServeStartCalls != 0 {
		t.Fatalf("ServeStarter must not Start; calls=%d", resp.ServeStartCalls)
	}
}
```
