## Expected

1. Exit 0.
2. `FocusCalled` with `FocusSession == sess-a`.
3. Output mentions focused (case-insensitive).
4. No `Error:` on stderr; no HTTP leak.
5. If stdout non-empty, ends with `\n`.

## Errors

- HTTP POST path; Focus hook not called; unknown command.

```go
import (
	"regexp"
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	if resp.HitHTTP {
		t.Fatalf("focus must not talk to daemon; out=%q err=%q", resp.Combined, resp.ErrMsg)
	}
	if !resp.FocusCalled {
		t.Fatal("Focus hook must be called")
	}
	if resp.FocusSession != "sess-a" {
		t.Fatalf("FocusSession=%q want sess-a", resp.FocusSession)
	}
	if !regexp.MustCompile(`(?i)focused`).MatchString(resp.Stdout + "\n" + resp.Stderr) {
		t.Fatalf("want focused confirmation; out=%q", resp.Combined)
	}
	if regexp.MustCompile(`Error:`).MatchString(resp.Stderr) {
		t.Fatalf("unexpected Error: %q", resp.Stderr)
	}
	if resp.Stdout != "" && !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatal("stdout must end with newline")
	}
}
```
