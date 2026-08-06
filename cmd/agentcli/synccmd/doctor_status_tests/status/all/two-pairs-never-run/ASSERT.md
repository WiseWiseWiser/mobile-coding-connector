## Expected

1. Harness err nil.
2. `StatusErr` empty.
3. Items include both `alpha` and `beta`.
4. Each LastRun contains `never`.
5. Item count is 2 (or at least both present without inventing a third).

## Side Effects

- None.

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
	if len(resp.Status.Items) != 2 {
		t.Fatalf("want 2 status items; got %d (%+v)", len(resp.Status.Items), resp.Status.Items)
	}
	for _, name := range []string{"alpha", "beta"} {
		it := statusNamed(resp.Status, name)
		if it == nil {
			t.Fatalf("missing status item %q; items=%+v", name, resp.Status.Items)
		}
		if !strings.Contains(strings.ToLower(it.LastRun), "never") {
			t.Fatalf("%s LastRun want never; got %q", name, it.LastRun)
		}
	}
}
```
