## Expected

- HTTP 200.
- Response includes `"running":true` and `"mocked":true`.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.APIStatusCode != 200 {
		t.Fatalf("status = %d body = %s", resp.APIStatusCode, resp.APIBody)
	}
	if !strings.Contains(resp.APIBody, `"running":true`) || !strings.Contains(resp.APIBody, `"mocked":true`) {
		t.Fatalf("unexpected start response: %s", resp.APIBody)
	}
}
```