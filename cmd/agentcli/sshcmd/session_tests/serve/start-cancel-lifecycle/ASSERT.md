## Expected

1. `ServeErr` is empty (context.Canceled after clean teardown is OK / not recorded as ServeErr).
2. `SessionAfterStart` is non-nil, `Alive` true, `LocalPort` > 0, `ProfileID` matches request.
3. `EchoThroughServe` equals EchoPayload (`"hi"`); `EchoErr` empty.
4. After cancel: `SessionAfterStop` is nil **or** `!Alive` (P1 gate fails).
5. `PortClosedAfterStop` is true (dial to former port fails).
6. `SSHConfigExists` is true; `SSHConfigMentionsPort` is true.

## Side Effects

- Session file written then cleared under `{Root}/ssh-sessions/`.
- `ssh_config` under ConfigDir during serve (may remain after stop; existence during start is required).

## Errors

None expected on the happy lifecycle path.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.ServeErr != "" {
		t.Fatalf("ServeService.Start error: %s", resp.ServeErr)
	}
	if resp.SessionAfterStart == nil {
		t.Fatal("SessionAfterStart is nil; Start must Save Alive session")
	}
	if !resp.SessionAfterStart.Alive {
		t.Fatal("SessionAfterStart.Alive must be true while serving")
	}
	if resp.SessionAfterStart.LocalPort <= 0 {
		t.Fatalf("SessionAfterStart.LocalPort: got %d want > 0", resp.SessionAfterStart.LocalPort)
	}
	if resp.SessionAfterStart.ProfileID != req.ProfileID {
		t.Fatalf("ProfileID: got %q want %q", resp.SessionAfterStart.ProfileID, req.ProfileID)
	}
	if resp.EchoErr != "" {
		t.Fatalf("echo through serve: %s", resp.EchoErr)
	}
	want := req.EchoPayload
	if want == "" {
		want = "hi"
	}
	if resp.EchoThroughServe != want {
		t.Fatalf("EchoThroughServe: got %q want %q", resp.EchoThroughServe, want)
	}
	// After stop: nil session or not Alive so P1 gate fails.
	if resp.SessionAfterStop != nil && resp.SessionAfterStop.Alive {
		t.Fatal("SessionAfterStop still Alive; cancel must Clear or mark not alive")
	}
	if !resp.PortClosedAfterStop {
		t.Fatal("listen port must be closed after cancel; dial still succeeded")
	}
	if !resp.SSHConfigExists {
		t.Fatal("ssh_config must exist under ConfigDir after Start")
	}
	if !resp.SSHConfigMentionsPort {
		t.Fatalf("ssh_config must mention LocalPort %d", resp.SessionAfterStart.LocalPort)
	}
}
```
