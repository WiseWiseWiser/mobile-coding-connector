## Expected

1. After unclean remote close, attach path reports failure: non-zero ExitCode
   and/or non-empty `AttachErr` (mentions close / lost / error / tty).
2. Does not hang.
3. Soft: if product wires `OnLocalRestore`, `TermRestored` may be true (not
   required for WS-only harness path).

## Errors

- Silent success with empty AttachErr and ExitCode 0 after unclean close.

## Exit Code

non-zero preferred

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	// Unclean close should surface as error to the operator path.
	if resp.ExitCode == 0 && strings.TrimSpace(resp.AttachErr) == "" {
		t.Fatalf("unclean disconnect must surface an error; ReceivedOutput=%q", resp.ReceivedOutput)
	}
	msg := strings.ToLower(resp.AttachErr + " " + resp.Combined)
	if msg != "" && !strings.Contains(msg, "close") && !strings.Contains(msg, "lost") &&
		!strings.Contains(msg, "error") && !strings.Contains(msg, "tty") &&
		!strings.Contains(msg, "disconnect") && !strings.Contains(msg, "100") {
		// Still accept any non-empty failure signal.
		if resp.ExitCode == 0 {
			t.Fatalf("unclear unclean-close error: %q", msg)
		}
	}
}
```
