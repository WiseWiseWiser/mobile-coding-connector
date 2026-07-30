---
explanation: "L2 service add requires --command"
---

## Expected

1. Non-zero exit.
2. Combined output mentions `command`.
3. No service created on disk.

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
		t.Fatalf("expected non-zero for missing --command; stdout:\n%s", resp.Stdout)
	}
	combined := strings.ToLower(resp.Combined)
	// Classic RED until `add` exists: unknown-subcommand must not count as success.
	if strings.Contains(combined, "unknown service subcommand") {
		t.Fatalf("service add not implemented yet (classic RED):\n%s", resp.Combined)
	}
	if !strings.Contains(combined, "--command") && !strings.Contains(combined, "command") {
		t.Fatalf("error should mention missing --command; combined:\n%s", resp.Combined)
	}
	if diskHasName(resp.ServicesOnDisk, "demo-no-cmd") {
		t.Fatalf("should not create demo-no-cmd without --command; disk=%v", resp.ServicesOnDisk)
	}
}
```
