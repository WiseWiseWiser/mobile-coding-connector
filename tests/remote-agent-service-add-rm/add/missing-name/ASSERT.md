---
explanation: "L2 service add requires --name"
---

## Expected

1. Non-zero exit.
2. Combined output mentions `name` (required flag / missing name).
3. No service named arbitrarily added (disk empty or unchanged empty).

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
		t.Fatalf("expected non-zero for missing --name; stdout:\n%s", resp.Stdout)
	}
	combined := strings.ToLower(resp.Combined)
	// Classic RED until `add` exists: unknown-subcommand must not count as success.
	if strings.Contains(combined, "unknown service subcommand") {
		t.Fatalf("service add not implemented yet (classic RED):\n%s", resp.Combined)
	}
	if !strings.Contains(combined, "--name") && !strings.Contains(combined, "name") {
		t.Fatalf("error should mention missing --name; combined:\n%s", resp.Combined)
	}
	if len(resp.ServicesOnDisk) > 0 {
		t.Fatalf("should not create services without --name; disk=%v", resp.ServicesOnDisk)
	}
}
```
