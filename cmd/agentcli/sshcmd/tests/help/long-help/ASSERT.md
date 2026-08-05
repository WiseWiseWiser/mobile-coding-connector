## Expected Output

Product help on stdout (minimum contract; implementer may add detail lines):

```
Usage: remote-agent ssh --serve
       remote-agent ssh [user@host] [command [args...]]
```

Must end with trailing `\n`. Assert checks key substrings (not a full v3
template) so implementers can expand help without rewriting leaves.

## Expected

1. `ParseErr` and `RunErr` are empty (nil errors).
2. Parsed `Mode` is `sshcmd.ModeHelp` (`"help"`).
3. Stdout contains `remote-agent ssh --serve`.
4. Stdout contains `user@host` (optional dest form).
5. Stdout contains `command` (command form).
6. Stdout ends with `\n`.
7. `ServeStartCalls` is 0; `RunnerCalls` is 0; `StoreLoadCalls` is 0.

## Side Effects

- No serve start, no runner invocation, no session load.

## Errors

- None expected.

## Exit Code

- Success path: returned error from `Run` is nil (library API; no process exit).

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
