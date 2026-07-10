## Expected

1. `ConfirmMessage` is exactly `Delete cron task "backup"?`.

## Errors

- Wrong punctuation, missing quotes, or generic "Are you sure?" copy.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := `Delete cron task "backup"?`
	if resp.ConfirmMessage != want {
		t.Fatalf("ConfirmMessage = %q, want %q", resp.ConfirmMessage, want)
	}
}
```
