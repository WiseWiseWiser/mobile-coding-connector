---
explanation: "L2 crontasks.Manager / agentcli"
---

## Expected

1. Run action succeeds.
2. History has ≥1 run within poll window (manual fire, not waiting ~1h).

## Side Effects

- One execution recorded; logs may contain `manual-fire`.

## Errors

- No history entry after manual run.

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
		t.Fatalf("manual run failed: %s", resp.ActionError)
	}
	if resp.RunCount < 1 {
		t.Fatalf("want ≥1 history run after manual fire, got %d", resp.RunCount)
	}
}
```
