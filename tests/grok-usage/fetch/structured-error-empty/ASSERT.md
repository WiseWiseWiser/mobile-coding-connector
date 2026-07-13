## Expected

1. `ServiceStatus` is `error`.
2. `ServiceError` is non-empty.
3. `WeeklyLimit` and `NextReset` are empty.
4. `ResetAt`, `ResetDisplay`, and `TimeLeft` are empty (no inventing).

## Errors

- Error status with fabricated structured countdown/display fields.

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
		t.Fatalf("raw usage fields should be empty on error: weekly=%q reset=%q",
			resp.WeeklyLimit, resp.NextReset)
	}
	if resp.ResetAt != "" || resp.ResetDisplay != "" || resp.TimeLeft != "" {
		t.Fatalf("structured fields must be empty on error: reset_at=%q reset_display=%q time_left=%q",
			resp.ResetAt, resp.ResetDisplay, resp.TimeLeft)
	}
}
```
