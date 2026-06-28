## Expected

- HTTP 200.
- Response body contains `"bot_token":"***"` and `"app_token":"***"`.
- Response must not contain raw token values.

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
	if !strings.Contains(resp.APIBody, `"bot_token":"***"`) || !strings.Contains(resp.APIBody, `"app_token":"***"`) {
		t.Fatalf("tokens not masked: %s", resp.APIBody)
	}
	if strings.Contains(resp.APIBody, "xoxb-secret") || strings.Contains(resp.APIBody, "xapp-secret") {
		t.Fatalf("raw secrets leaked in response: %s", resp.APIBody)
	}
}
```