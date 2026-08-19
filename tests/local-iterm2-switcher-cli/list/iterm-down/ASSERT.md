## Expected

1. Exit 0 (warning, not a missing command).
2. Combined output includes `iTerm2 is not running` (`FormatEmptyITerm`).
3. No HTTP leak.

## Errors

- `unknown command: native-terminals`; hard error without the formatter copy; daemon dial.

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
	if resp.ExitCode != 0 {
		t.Fatalf("iterm-down is a warning; exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	if resp.HitHTTP {
		t.Fatalf("iterm-down must not dial daemon; out=%q err=%q", resp.Combined, resp.ErrMsg)
	}
	if !regexp.MustCompile(`iTerm2 is not running`).MatchString(resp.Stdout + "\n" + resp.Stderr) {
		t.Fatalf("want FormatEmptyITerm; out=%q", resp.Combined)
	}
}
```
