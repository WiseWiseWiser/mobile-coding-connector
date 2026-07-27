---
label: slow, negative, e2e
explanation: never-respond fake TUI subprocess; expects error
---
## Expected

When the fake TUI never renders parseable `/status` output, fetch must fail cleanly:

1. `ServiceStatus` is `error`.
2. `ServiceError` contains `timeout waiting for status output`.
3. `MonthlyUsage` is empty.

## Errors

- `status=ready` (must not succeed without parseable output).
- Empty or unrelated error message.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServiceStatus != "error" {
		t.Fatalf("status = %q error = %q, want error", resp.ServiceStatus, resp.ServiceError)
	}
	if !strings.Contains(resp.ServiceError, "timeout waiting for status output") {
		t.Fatalf("service error = %q, want timeout waiting for status output", resp.ServiceError)
	}
	if strings.TrimSpace(resp.MonthlyUsage) != "" {
		t.Fatalf("monthly_usage = %q, want empty on error", resp.MonthlyUsage)
	}
}
```
