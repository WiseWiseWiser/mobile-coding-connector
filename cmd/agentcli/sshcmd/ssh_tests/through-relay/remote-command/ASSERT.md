## Expected

1. Harness `err` is nil.
2. `ServeErr` empty.
3. `SessionAfterStart` non-nil, Alive, LocalPort > 0.
4. `RunnerErr` empty.
5. `Stdout` contains `EchoNeedle` (`hello`).
6. `RelayLocalPort` > 0.

## Side Effects

- Session file under `{Root}/ssh-sessions/` during serve; cleared on cancel.
- Adhoc + relay closed after Run.

## Errors

None expected on happy path.

```go
import (
	"strings"
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
		t.Fatalf("ServeService error: %s", resp.ServeErr)
	}
	if resp.SessionAfterStart == nil || !resp.SessionAfterStart.Alive {
		t.Fatal("SessionAfterStart must be Alive with LocalPort after Start")
	}
	if resp.SessionAfterStart.LocalPort <= 0 {
		t.Fatalf("SessionAfterStart.LocalPort: got %d", resp.SessionAfterStart.LocalPort)
	}
	if resp.RunnerErr != "" {
		t.Fatalf("CryptoSSHRunner through relay: %s (stderr=%q)", resp.RunnerErr, resp.Stderr)
	}
	if resp.RelayLocalPort <= 0 {
		t.Fatalf("RelayLocalPort: got %d want > 0", resp.RelayLocalPort)
	}
	needle := req.EchoNeedle
	if needle == "" {
		needle = "hello"
	}
	if !strings.Contains(resp.Stdout, needle) {
		t.Fatalf("Stdout must contain %q; got %q", needle, resp.Stdout)
	}
}
```
