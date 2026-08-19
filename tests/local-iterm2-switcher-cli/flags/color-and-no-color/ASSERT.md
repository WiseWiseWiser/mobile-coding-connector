## Expected

1. Non-zero exit.
2. Error: `--color and --no-color cannot be specified together`.

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
		t.Fatal("--color --no-color must be non-zero")
	}
	text := resp.Stderr + "\n" + resp.ErrMsg
	if !regexp.MustCompile(`Error:`).MatchString(resp.Stderr) {
		t.Fatalf("want Error: prefix; stderr=%q", resp.Stderr)
	}
	if !regexp.MustCompile(`--color and --no-color cannot be specified together`).MatchString(text) {
		t.Fatalf("want color conflict message; err=%q", text)
	}
}
```
