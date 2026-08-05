## Expected

1. `ParseErr` and `RunErr` are empty.
2. Parsed `Mode` is `sshcmd.ModeHelp`.
3. Stdout contains `remote-agent ssh --serve`, `user@host`, and `command`.
4. Stdout ends with `\n`.
5. No ServeStarter / SSHRunner / SessionStore calls.

## Side Effects

None.

## Errors

None expected.

```go
import (
	"strings"
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
	if resp.Mode != sshcmd.ModeHelp {
		t.Fatalf("Mode: got %q want %q", resp.Mode, sshcmd.ModeHelp)
	}
	if !strings.Contains(resp.Stdout, "remote-agent ssh --serve") {
		t.Fatalf("stdout missing --serve usage; got:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "user@host") {
		t.Fatalf("stdout missing user@host form; got:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "command") {
		t.Fatalf("stdout missing command form; got:\n%s", resp.Stdout)
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("stdout must end with trailing newline; got %q", resp.Stdout)
	}
	if resp.ServeStartCalls != 0 || resp.RunnerCalls != 0 || resp.StoreLoadCalls != 0 {
		t.Fatalf("help must not call deps: serve=%d runner=%d store=%d",
			resp.ServeStartCalls, resp.RunnerCalls, resp.StoreLoadCalls)
	}
}
```
