---
explanation: "L2 service rm by id"
---

## Expected

1. Exit 0.
2. Stdout contains Removed and id (and/or name).
3. ListAll / disk no longer contain `svc-rm-id-001`.

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
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(strings.ToLower(resp.Stdout), "removed") {
		t.Fatalf("stdout missing Removed:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "svc-rm-id-001") &&
		!strings.Contains(resp.Stdout, "rm-by-id-target") {
		t.Fatalf("stdout should mention id or name:\n%s", resp.Stdout)
	}
	if listContainsID(resp.ListedIDs, "svc-rm-id-001") {
		t.Fatalf("ListAll still has svc-rm-id-001: %v", resp.ListedIDs)
	}
	if diskHasID(resp.ServicesOnDisk, "svc-rm-id-001") {
		t.Fatalf("disk still has svc-rm-id-001: %v", resp.ServicesOnDisk)
	}
}
```
