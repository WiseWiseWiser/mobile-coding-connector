---
explanation: "L2 service list --project-dir hides other project"
---

## Expected

1. Exit 0.
2. Stdout contains local name/id (`web` / `local-web`).
3. Stdout does **not** contain other name/id as a listed service
   (`api` / `other-api` — careful with substring; prefer id `other-api`).
4. Manager ListAll still has both (seed intact); CLI output is scoped.

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
	if !strings.Contains(out, "web") && !strings.Contains(out, "local-web") {
		t.Fatalf("scoped list missing local service; stdout:\n%s", out)
	}
	// Prefer id check to avoid matching "api" inside unrelated words.
	if strings.Contains(out, "other-api") {
		t.Fatalf("scoped list should hide other-api; stdout:\n%s", out)
	}
	// ListAll snapshot still has both seeds (manager unscoped).
	if !listContainsID(resp.ListedIDs, "local-web") || !listContainsID(resp.ListedIDs, "other-api") {
		t.Fatalf("manager ListAll should still hold both seeds: %v", resp.ListedIDs)
	}
}
```
