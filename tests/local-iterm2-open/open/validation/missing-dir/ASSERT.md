## Expected

1. Status in 400–499.
2. `Error` non-empty (JSON `error` field).
3. Open ideally not called (if called, still fail status assert above).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 400 || resp.StatusCode > 499 {
		t.Fatalf("status = %d, want 4xx; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Error == "" {
		t.Fatalf("want non-empty error field; body=%s", resp.Body)
	}
}
```
