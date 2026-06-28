## Expected

- First start: HTTP 200.
- Second start: HTTP 409.
- Error code `ALREADY_RUNNING`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.APIStatusCode != 409 {
		t.Fatalf("second start status = %d, want 409 body = %s", resp.APIStatusCode, resp.APIBody)
	}
	if apiErrorCode(resp.APIBody) != "ALREADY_RUNNING" {
		t.Fatalf("error code = %s, want ALREADY_RUNNING", apiErrorCode(resp.APIBody))
	}
}
```