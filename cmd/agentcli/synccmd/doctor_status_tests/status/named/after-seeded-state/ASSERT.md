## Expected

1. Harness err nil.
2. `StatusErr` empty.
3. Item `mad-max` present.
4. `LastRun` contains `2026-01-15` (or full timestamp); must not be only `never`.
5. Optional: LastExit is 0 when populated.

## Side Effects

- State file read only.

## Errors

- None.

## Exit Code

- Nil library error.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.StatusErr != "" {
		t.Fatalf("Status error: %s", resp.StatusErr)
	}
	it := statusNamed(resp.Status, "mad-max")
	if it == nil {
		t.Fatalf("missing status item mad-max; items=%+v", resp.Status.Items)
	}
	if !strings.Contains(it.LastRun, "2026-01-15") {
		t.Fatalf("LastRun should reflect seeded lastRunAt; got %q", it.LastRun)
	}
	if strings.EqualFold(strings.TrimSpace(it.LastRun), "never") {
		t.Fatalf("LastRun still never despite state file: %q", it.LastRun)
	}
}
```
