## Expected

1. Exit 0.
2. Stdout documents `port list` and mentions `--json` and/or `--forwards`.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	if !strings.Contains(out, "list") {
		t.Fatalf("expected list help; stdout:\n%s", out)
	}
	if !strings.Contains(out, "--json") && !strings.Contains(out, "json") {
		t.Fatalf("list help should document --json; stdout:\n%s", out)
	}
}
```
