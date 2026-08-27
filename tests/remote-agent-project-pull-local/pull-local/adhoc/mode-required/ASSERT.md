## Expected

- Exit code 1.
- Output mentions `--mode` required / git-fetch or download.

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
	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "--mode") {
		t.Fatalf("expected --mode required message;\n%s", resp.Combined)
	}
}
```
