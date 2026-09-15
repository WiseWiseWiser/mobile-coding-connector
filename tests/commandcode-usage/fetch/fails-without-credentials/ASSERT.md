## Expected

1. Status is `error` and the message says the account is not authenticated.
2. No panel and no usage URL are invented.
3. `updated_at` is still stamped.

## Errors

- Returning a ready response with empty fields, or dropping the timestamp.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
	if !strings.Contains(resp.Error, "not authenticated") {
		t.Fatalf("error = %q", resp.Error)
	}
	if resp.Detail != "" || resp.UsageURL != "" {
		t.Fatalf("error response must not carry a panel: %q / %q", resp.Detail, resp.UsageURL)
	}
	if resp.UpdatedAt == "" {
		t.Fatal("an error response must stamp updated_at")
	}
}
```
