## Expected

- HTTP 400.
- Error code `BAD_REQUEST`.
- Message mentions bot token.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.APIStatusCode != 400 {
		t.Fatalf("status = %d, want 400 body = %s", resp.APIStatusCode, resp.APIBody)
	}
	if apiErrorCode(resp.APIBody) != "BAD_REQUEST" {
		t.Fatalf("error code = %s, want BAD_REQUEST", apiErrorCode(resp.APIBody))
	}
	if !strings.Contains(resp.APIBody, "bot token") {
		t.Fatalf("expected bot token message: %s", resp.APIBody)
	}
}
```