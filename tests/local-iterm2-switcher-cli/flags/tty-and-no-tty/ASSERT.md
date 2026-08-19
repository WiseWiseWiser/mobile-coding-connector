## Expected

1. Non-zero exit.
2. Error mentions both flags cannot be specified together.

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
		t.Fatal("--tty --no-tty must be non-zero")
	}
	text := resp.Stderr + "\n" + resp.ErrMsg
	if !regexp.MustCompile(`Error:`).MatchString(resp.Stderr) {
		t.Fatalf("want Error: prefix; stderr=%q", resp.Stderr)
	}
	if !regexp.MustCompile(`--tty and --no-tty cannot be specified together`).MatchString(text) {
		t.Fatalf("want conflict message; err=%q", text)
	}
}
```
