---
label: slow
explanation: writes 65 MiB of fixture data on remote project dir
---

## Expected

1. Exit code 1.
2. Combined output suggests raising limit via `--max-size`.

## Side Effects

Remote bulk files remain; no successful full pull worktree.

## Errors

- Exit 0.

## Exit Code

1.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "max-size") && !strings.Contains(lower, "max size") {
		t.Fatalf("expected --max-size hint;\n%s", resp.Combined)
	}
	if !strings.Contains(lower, "64") && !strings.Contains(lower, "size") && !strings.Contains(lower, "limit") {
		t.Fatalf("expected total size limit message;\n%s", resp.Combined)
	}
}
```