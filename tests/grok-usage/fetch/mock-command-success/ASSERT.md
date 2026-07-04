## Expected

1. `ServiceStatus` is `ready`.
2. `WeeklyLimit` is `6%`.
3. `NextReset` is `July 9, 16:55 PT`.
4. `UpdatedAt` is non-empty RFC3339 timestamp.

## Errors

- Service reports loading/error or missing fields.

```go
import (
	"testing"
	"time"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServiceStatus != "ready" {
		t.Fatalf("status = %q, want ready", resp.ServiceStatus)
	}
	if resp.WeeklyLimit != "6%" {
		t.Fatalf("weekly_limit = %q", resp.WeeklyLimit)
	}
	if resp.NextReset != "July 9, 16:55 PT" {
		t.Fatalf("next_reset = %q", resp.NextReset)
	}
	if resp.UpdatedAt == "" {
		t.Fatal("updated_at empty")
	}
	if _, parseErr := time.Parse(time.RFC3339, resp.UpdatedAt); parseErr != nil {
		t.Fatalf("updated_at not RFC3339: %v", parseErr)
	}
}
```