---
explanation: "L2 plain service list is global"
---

## Expected

1. Exit 0.
2. Stdout contains both local and other name/id (`web` / `local-web` and `api` / `other-api`).
3. Manager List has both seeds.

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
		t.Fatalf("plain list should show both services; stdout:\n%s", out)
	}
	if !listContainsID(resp.ListedIDs, "local-web") || !listContainsID(resp.ListedIDs, "other-api") {
		t.Fatalf("manager List should hold both seeds: %v", resp.ListedIDs)
	}
}
```
