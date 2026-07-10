## Expected

1. Status `401`.
2. Body has error field (unauthorized / not_initialized depending on credentials wiring).
3. Open not called.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.OpenCalled {
		t.Fatal("Open must not run without auth")
	}
}
```
