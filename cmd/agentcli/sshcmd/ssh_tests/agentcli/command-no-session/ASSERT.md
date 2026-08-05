## Expected

1. Harness `err` is nil (product error is in `AgentcliErr`).
2. `UnknownCommand` is false.
3. `AgentcliErr` contains
   `no active SSH tunnel; run 'remote-agent ssh --serve' first`.

## Side Effects

- None required (missing session file is fine).

## Errors

- Expected product error: tunnel not active.

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
	if resp.UnknownCommand {
		t.Fatalf("ssh must not be unknown command; AgentcliErr=%q", resp.AgentcliErr)
	}
	if resp.AgentcliErr == "" {
		t.Fatal("expected error for ssh ls without active tunnel; got nil")
	}
	if !strings.Contains(resp.AgentcliErr, sshcmd.ErrNoActiveTunnel) {
		t.Fatalf("error must contain %q; got %q", sshcmd.ErrNoActiveTunnel, resp.AgentcliErr)
	}
}
```
