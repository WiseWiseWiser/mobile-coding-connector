## Expected

1. Non-zero exit.
2. `Error:` prefix.
3. Canonical form: `unknown native-terminals subcommand: foo` (not the typed alias).

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
		t.Fatal("unknown subcommand must be non-zero")
	}
	text := resp.Stderr + "\n" + resp.ErrMsg
	if !regexp.MustCompile(`Error:`).MatchString(resp.Stderr) {
		t.Fatalf("want Error: prefix; stderr=%q", resp.Stderr)
	}
	if !regexp.MustCompile(`unknown native-terminals subcommand: foo`).MatchString(text) {
		t.Fatalf("want canonical unknown native-terminals subcommand: foo; err=%q", text)
	}
}
```
