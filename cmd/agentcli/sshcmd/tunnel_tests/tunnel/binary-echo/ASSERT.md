## Expected

1. Harness `err` is nil.
2. `CreateErr` empty; `SessionID` non-empty.
3. `TunnelDialErr` empty.
4. `EchoErr` empty.
5. `EchoRead` equals `EchoWrote` (or at least has EchoWrote as prefix when extra bytes).

## Side Effects

- One WS + one TCP to echo backend; closed after Run.

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
	if resp.CreateErr != "" {
		t.Fatalf("CreateSSHSession: %s", resp.CreateErr)
	}
	if resp.SessionID == "" {
		t.Fatal("SessionID required before tunnel dial")
	}
	if resp.TunnelDialErr != "" {
		t.Fatalf("SSHTunnelDial: %s", resp.TunnelDialErr)
	}
	if resp.EchoErr != "" {
		t.Fatalf("echo I/O: %s", resp.EchoErr)
	}
	want := req.EchoPayload
	if want == "" {
		want = "p4-tunnel-hi"
	}
	if resp.EchoWrote != want {
		t.Fatalf("EchoWrote: got %q want %q", resp.EchoWrote, want)
	}
	if !strings.HasPrefix(resp.EchoRead, want) && resp.EchoRead != want {
		t.Fatalf("EchoRead must equal/prefix payload %q; got %q", want, resp.EchoRead)
	}
}
```
