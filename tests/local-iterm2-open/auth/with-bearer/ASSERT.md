## Expected

1. Status `200` with `ok:true`.
2. Open called.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, resp.Body)
	}
	if !resp.OK {
		t.Fatalf("want ok:true; body=%s", resp.Body)
	}
	if !resp.OpenCalled {
		t.Fatal("Open not called with valid Bearer")
	}
}
```
