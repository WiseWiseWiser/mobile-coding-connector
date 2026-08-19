## Expected

1. Non-zero exit.
2. `Error:` on stderr.
3. `FocusCalled == false` (no Focus hook without an id).
4. Not `unknown command: native-terminals`; usage/session wording.

## Errors

- Command missing; Focus called with empty id.

```go
import (
	"regexp"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatal("missing session-id must be non-zero")
	}
	if !regexp.MustCompile(`Error:`).MatchString(resp.Stderr) {
		t.Fatalf("want Error: prefix; stderr=%q", resp.Stderr)
	}
	if resp.FocusCalled {
		t.Fatalf("must not call Focus without session-id; session=%q", resp.FocusSession)
	}
	text := resp.Stderr + "\n" + resp.ErrMsg
	if regexp.MustCompile(`unknown command: native-terminals`).MatchString(text) {
		t.Fatal("native-terminals command must exist; missing-arg is a usage error")
	}
	if !regexp.MustCompile(`(?i)session|usage|required`).MatchString(text) {
		t.Fatalf("want session-id usage error; err=%q", text)
	}
}
```
