## Expected

- Both stop calls succeed.
- `Status.Running` remains false.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.Status.Running {
		t.Fatalf("unexpected running after idempotent stop: %+v", resp.Status)
	}
}
```