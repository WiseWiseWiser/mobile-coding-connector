---
explanation: "L2 service list --all cross-project"
---

## Expected

1. Exit 0.
2. Stdout contains both service names `web` and `api` (or ids `local-web` / `other-api`).
3. Manager ListAll snapshot includes both ids.

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
	out := resp.Stdout
	hasWeb := strings.Contains(out, "web") || strings.Contains(out, "local-web")
	hasAPI := strings.Contains(out, "api") || strings.Contains(out, "other-api")
	if !hasWeb || !hasAPI {
		t.Fatalf("list --all stdout should show both scopes; stdout:\n%s", out)
	}
	if !listContainsID(resp.ListedIDs, "local-web") || !listContainsID(resp.ListedIDs, "other-api") {
		t.Fatalf("ListAll snapshot missing seeds: %v", resp.ListedIDs)
	}
}
```
