---
explanation: "L2 service rm by name"
---

## Expected

1. Exit 0.
2. Stdout contains `Removed` (case-insensitive) and name or id.
3. Stdout ends with newline.
4. ListAll does not contain `svc-rm-name-001` / `rm-by-name-target`.
5. Disk does not have that id.

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
	out := strings.ToLower(resp.Stdout)
	if !strings.Contains(out, "removed") {
		t.Fatalf("stdout missing Removed:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "rm-by-name-target") &&
		!strings.Contains(resp.Stdout, "svc-rm-name-001") {
		t.Fatalf("stdout should mention name or id:\n%s", resp.Stdout)
	}
	if !strings.HasSuffix(resp.Stdout, "\n") {
		t.Fatalf("stdout must end with newline; got %q", resp.Stdout)
	}
	if listContainsID(resp.ListedIDs, "svc-rm-name-001") {
		t.Fatalf("ListAll still has svc-rm-name-001: %v", resp.ListedIDs)
	}
	if listContainsName(resp.ListedNames, "rm-by-name-target") {
		t.Fatalf("ListAll still has rm-by-name-target: %v", resp.ListedNames)
	}
	if diskHasID(resp.ServicesOnDisk, "svc-rm-name-001") {
		t.Fatalf("disk still has svc-rm-name-001: %v", resp.ServicesOnDisk)
	}
}
```
