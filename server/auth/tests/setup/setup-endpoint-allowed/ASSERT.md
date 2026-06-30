## Expected

`POST /api/auth/setup` on uninitialized server must pass through middleware:

1. `Response.StatusCode` is not `401` with `not_initialized`.
2. `Response.JSON["error"]` is not `"not_initialized"`.
3. Handler processes the request (200 with status ok, or 400 for validation).

## Errors

- Middleware blocks with `not_initialized` (would indicate skip-path regression).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if resp.JSON != nil {
		if errVal, _ := resp.JSON["error"].(string); errVal == "not_initialized" {
			t.Fatalf("POST /api/auth/setup blocked with not_initialized: %s", resp.Body)
		}
	}

	if resp.StatusCode == 401 && resp.Body == `{"error":"not_initialized"}`+"\n" {
		t.Fatalf("setup endpoint must be in skip paths; got: %s", resp.Body)
	}

	t.Logf("setup endpoint reached handler: status=%d body=%s", resp.StatusCode, resp.Body)
}
```