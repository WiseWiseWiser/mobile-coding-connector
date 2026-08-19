## Expected

1. Non-zero exit.
2. `unknown command: terminals` (bare name removed; not an alias).

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
		t.Fatal("bare terminals must be unknown")
	}
	text := resp.Stderr + "\n" + resp.ErrMsg
	if !regexp.MustCompile(`unknown command: terminals`).MatchString(text) {
		t.Fatalf("want unknown command: terminals; err=%q", text)
	}
}
```
