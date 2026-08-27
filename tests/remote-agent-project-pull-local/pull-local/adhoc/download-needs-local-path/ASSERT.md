## Expected

- Exit code 1.
- Output mentions `--local-path`.

## Exit Code

1.

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
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}
	if !strings.Contains(strings.ToLower(resp.Combined), "local-path") {
		t.Fatalf("expected --local-path required;\n%s", resp.Combined)
	}
}
```
