## Expected

1. `ServiceStatus` is `error`.
2. `ServiceError` is non-empty.
3. `WeeklyLimit` and `NextReset` are empty.

## Errors

- Service reports ready on failing script.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServiceStatus != "error" {
		t.Fatalf("status = %q, want error", resp.ServiceStatus)
	}
	if resp.ServiceError == "" {
		t.Fatal("error message empty")
	}
	if resp.WeeklyLimit != "" || resp.NextReset != "" {
		t.Fatalf("limits should be empty on error: weekly=%q reset=%q", resp.WeeklyLimit, resp.NextReset)
	}
}
```