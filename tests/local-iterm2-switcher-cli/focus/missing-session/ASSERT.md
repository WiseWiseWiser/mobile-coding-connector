## Expected

1. Non-zero exit.
2. Stderr has `Error:` prefix.
3. Not merely `unknown command: native-terminals` — session/not-found wording or Focus attempt that failed.
4. No successful "focused" confirmation.

## Errors

- Command missing; silent success.

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
		t.Fatal("unknown session must be non-zero")
	}
	if !regexp.MustCompile(`Error:`).MatchString(resp.Stderr) {
		t.Fatalf("want Error: prefix; stderr=%q", resp.Stderr)
	}
	text := resp.Stderr + "\n" + resp.ErrMsg
	if regexp.MustCompile(`unknown command: native-terminals`).MatchString(text) &&
		!regexp.MustCompile(`(?i)no-such-session|not found|session`).MatchString(text) {
		t.Fatalf("want session error, not missing command only; err=%q", text)
	}
	if !regexp.MustCompile(`(?i)no-such-session|not found|session`).MatchString(text) {
		t.Fatalf("want session-not-found style error; err=%q", text)
	}
	if regexp.MustCompile(`(?i)focused`).MatchString(resp.Stdout) {
		t.Fatal("must not print focused on unknown session")
	}
}
```
