---
explanation: "L2 service rm cross-projectDir resolution via ListAll"
---

## Expected

1. Exit 0 (resolves name even though projectDir ≠ default scope).
2. Stdout contains Removed and name or id.
3. ListAll / disk no longer contain `svc-cross-001`.

## Errors

- Non-zero with "no service found" — typical pre-fix when resolution is scoped.

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
		t.Fatalf("Run error: %v\ncombined:\n%s", err, resp.Combined)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d (cross-scope resolve failed?); combined:\n%s",
			resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(strings.ToLower(resp.Stdout), "removed") {
		t.Fatalf("stdout missing Removed:\n%s", resp.Stdout)
	}
	if listContainsID(resp.ListedIDs, "svc-cross-001") {
		t.Fatalf("ListAll still has svc-cross-001: %v", resp.ListedIDs)
	}
	if diskHasID(resp.ServicesOnDisk, "svc-cross-001") {
		t.Fatalf("disk still has svc-cross-001: %v", resp.ServicesOnDisk)
	}
}
```
