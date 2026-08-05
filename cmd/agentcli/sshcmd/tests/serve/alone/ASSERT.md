## Expected

1. `ParseErr` and `RunErr` are empty.
2. Parsed `Mode` is `sshcmd.ModeServe`.
3. `ServeStartCalls` is 1; `ServeProfileID` equals Request profile (`default`).
4. `RunnerCalls` is 0.
5. Session store is not required for serve (Load may be 0).

## Side Effects

- Exactly one `ServeStarter.Start` invocation recorded by the mock.

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
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("Parse error: %s", resp.ParseErr)
	}
	if resp.RunErr != "" {
		t.Fatalf("sshcmd.Run error: %s", resp.RunErr)
	}
	if resp.Mode != sshcmd.ModeServe {
		t.Fatalf("Mode: got %q want %q", resp.Mode, sshcmd.ModeServe)
	}
	if resp.ServeStartCalls != 1 {
		t.Fatalf("ServeStarter.Start calls: got %d want 1", resp.ServeStartCalls)
	}
	wantProfile := req.ProfileID
	if wantProfile == "" {
		wantProfile = "default"
	}
	if resp.ServeProfileID != wantProfile {
		t.Fatalf("Serve ProfileID: got %q want %q", resp.ServeProfileID, wantProfile)
	}
	if resp.RunnerCalls != 0 {
		t.Fatalf("SSHRunner must not be called in serve mode; calls=%d", resp.RunnerCalls)
	}
}
```
