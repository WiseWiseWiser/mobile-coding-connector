## Expected

1. `ParseErr` and `RunErr` empty.
2. `Mode` is `sshcmd.ModeCommand`.
3. Parsed and Runner remote argv are `["uname","-a"]`.
4. `RunnerCalls` is 1; `ServeStartCalls` is 0.

## Side Effects

- One SSHRunner.Run with multi-arg remote list.

## Errors

None expected.

```go
import (
	"reflect"
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
	if resp.Mode != sshcmd.ModeCommand {
		t.Fatalf("Mode: got %q want %q", resp.Mode, sshcmd.ModeCommand)
	}
	want := []string{"uname", "-a"}
	if !reflect.DeepEqual(resp.RemoteArgv, want) {
		t.Fatalf("parsed RemoteArgv: got %v want %v", resp.RemoteArgv, want)
	}
	if resp.RunnerCalls != 1 {
		t.Fatalf("SSHRunner.Run calls: got %d want 1", resp.RunnerCalls)
	}
	if !reflect.DeepEqual(resp.RunnerRemoteArgv, want) {
		t.Fatalf("Runner remote argv: got %v want %v", resp.RunnerRemoteArgv, want)
	}
	if resp.ServeStartCalls != 0 {
		t.Fatalf("ServeStarter must not Start; calls=%d", resp.ServeStartCalls)
	}
}
```
