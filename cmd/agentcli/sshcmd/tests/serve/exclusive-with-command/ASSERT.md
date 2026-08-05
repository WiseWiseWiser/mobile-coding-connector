## Expected

1. Combined error text (ParseErr or RunErr) is non-empty.
2. Error mentions `--serve` and that it cannot be combined with a remote command
   (substring `cannot` and `--serve`, or equivalent exclusive wording including
   `command`).
3. `ServeStartCalls` is 0.
4. `RunnerCalls` is 0.

## Side Effects

- No serve start and no runner call on exclusive failure.

## Errors

- Expected failure: exclusive `--serve` + remote command.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	msg := errText(resp)
	if msg == "" {
		t.Fatalf("expected exclusive error for --serve + command; got success")
	}
	lower := strings.ToLower(msg)
	if !strings.Contains(msg, "--serve") {
		t.Fatalf("error should mention --serve; got %q", msg)
	}
	if !strings.Contains(lower, "command") && !strings.Contains(lower, "cannot") && !strings.Contains(lower, "exclusive") {
		t.Fatalf("error should signal exclusive serve/command conflict; got %q", msg)
	}
	if resp.ServeStartCalls != 0 {
		t.Fatalf("ServeStarter must not Start on exclusive error; calls=%d", resp.ServeStartCalls)
	}
	if resp.RunnerCalls != 0 {
		t.Fatalf("SSHRunner must not Run on exclusive error; calls=%d", resp.RunnerCalls)
	}
}
```
