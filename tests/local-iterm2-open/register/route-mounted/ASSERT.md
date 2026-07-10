## Expected

1. Status is not 404.
2. Status `200` with `ok:true` when Open inject succeeds.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == 404 {
		t.Fatalf("route not mounted: status 404 body=%s", resp.Body)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 after Register; body=%s", resp.StatusCode, resp.Body)
	}
	if !resp.OK {
		t.Fatalf("want ok:true; body=%s", resp.Body)
	}
	if !resp.OpenCalled {
		t.Fatal("Open not called via registered route")
	}
}
```
