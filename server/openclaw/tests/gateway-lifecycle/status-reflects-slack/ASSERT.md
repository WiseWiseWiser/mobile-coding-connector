## Expected

- `Status.SlackEnabled` is true.
- `Status.SlackMode` is `socket`.
- `Status.Mocked` is always true.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !resp.Status.SlackEnabled || resp.Status.SlackMode != "socket" {
		t.Fatalf("status = %+v", resp.Status)
	}
	if !resp.Status.Mocked {
		t.Fatal("status.mocked should be true")
	}
}
```