## Expected

1. `AlertMessage` is exactly `Task disabled until next schedule`.

## Errors

- Fallback used when server message is present; trimming dropped content incorrectly.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Task disabled until next schedule"
	if resp.AlertMessage != want {
		t.Fatalf("alert = %q, want %q", resp.AlertMessage, want)
	}
}
```
