---
explanation: "L2 crontasks.Manager / agentcli"
---

## Expected

1. `Run` returns without harness error.
2. `Response.Tasks` is empty (length 0).
3. No `ActionError`.

## Side Effects

- None (read-only list).

## Errors

- Non-empty list on fresh home.
- HTTP/list failure.

## Exit Code

0 from `Run`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("unexpected ActionError: %s", resp.ActionError)
	}
	if len(resp.Tasks) != 0 {
		t.Fatalf("want empty list, got %d tasks: %+v", len(resp.Tasks), resp.Tasks)
	}
}
```
