---
explanation: "L2 service rm by name via global list"
---

## Expected

1. Exit 0.
2. Stdout contains Removed / cross-scope-svc (or id).
3. List / disk no longer contain `svc-cross-001`.

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
	out := resp.Stdout + resp.Stderr
	if !strings.Contains(out, "Removed") {
		t.Fatalf("want Removed in output; got:\n%s", out)
	}
	if listContainsID(resp.ListedIDs, "svc-cross-001") {
		t.Fatalf("List still has svc-cross-001: %v", resp.ListedIDs)
	}
	if diskHasID(resp.ServicesOnDisk, "svc-cross-001") {
		t.Fatalf("disk still has svc-cross-001")
	}
}
```
