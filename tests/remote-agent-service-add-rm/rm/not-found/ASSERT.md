---
explanation: "L2 service rm unknown target"
---

## Expected

1. Non-zero exit.
2. Combined mentions not found / no service (or the target name).

## Exit Code

non-zero.

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
		t.Fatalf("expected non-zero for unknown rm target; stdout:\n%s", resp.Stdout)
	}
	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "not found") &&
		!strings.Contains(combined, "no service") &&
		!strings.Contains(combined, "does-not-exist-svc") {
		t.Fatalf("error should mention missing target; combined:\n%s", resp.Combined)
	}
}
```
