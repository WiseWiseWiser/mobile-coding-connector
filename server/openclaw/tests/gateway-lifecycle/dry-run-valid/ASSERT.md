## Expected

- HTTP 200.
- Body includes `"mocked":true`.
- Checks mention mocked gateway and slack socket.

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
	if !strings.Contains(resp.APIBody, `"mocked":true`) {
		t.Fatalf("dry_run should be mocked: %s", resp.APIBody)
	}
	if !strings.Contains(resp.APIBody, "gateway integration is mocked") {
		t.Fatalf("missing gateway mock check: %s", resp.APIBody)
	}
	if !strings.Contains(resp.APIBody, "slack socket mode connection is mocked") {
		t.Fatalf("missing slack mock check: %s", resp.APIBody)
	}
}
```