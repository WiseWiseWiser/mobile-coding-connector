## Expected

- `Status.Running` is false after stop.
- `State.Running` is false.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.Status.Running || resp.State.Running {
		t.Fatalf("still running: status=%+v state=%+v", resp.Status, resp.State)
	}
}
```