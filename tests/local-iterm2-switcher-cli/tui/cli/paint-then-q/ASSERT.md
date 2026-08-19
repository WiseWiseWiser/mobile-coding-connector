## Expected

1. Exit 0.
2. Stdout (ANSI-stripped) has split box, `grok review`, and status `cached` or `up to date` or `incremental`.
3. FocusCalled=false.
4. CaptureCalls=0 when layout unchanged.

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
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	view := resp.ViewText
	if !regexp.MustCompile(`┌|│|Terminals`).MatchString(view) {
		t.Fatalf("--tty must paint split TUI; stdout=%q", resp.Stdout)
	}
	if !regexp.MustCompile(`grok review`).MatchString(view) {
		t.Fatalf("missing session primary; stdout=%q", resp.Stdout)
	}
	if !regexp.MustCompile(`cached|up to date|incremental`).MatchString(view) {
		t.Fatalf("missing status token; stdout=%q", resp.Stdout)
	}
	if resp.FocusCalled {
		t.Fatal("q must not focus a session")
	}
	if resp.CaptureCalls != 0 {
		t.Fatalf("CaptureCalls=%d want 0 (warm same layout)", resp.CaptureCalls)
	}
}
```
