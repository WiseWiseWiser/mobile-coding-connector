## Expected

1. `AlertMessage` is exactly `The server won't stop immediately unless you manually stop it`.

## Errors

- Alert copy diverges from `server/services` `msgDisableRunning`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AlertMessage != msgDisableRunning {
		t.Fatalf("alert = %q, want %q", resp.AlertMessage, msgDisableRunning)
	}
}
```