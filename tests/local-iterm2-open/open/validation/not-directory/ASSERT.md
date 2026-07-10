## Expected

1. Status **4xx** (validation — not a directory). Prefer 400.
2. Non-empty `error` field.
3. Prefer Open not called (handler validates before Open); if Open is called and fails, still must surface as 4xx for this case.

## Errors

- Returning 5xx for client path validation (use 5xx only for unexpected Open/osascript failures after a valid dir).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 400 || resp.StatusCode > 499 {
		t.Fatalf("status = %d, want 4xx for non-directory; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Error == "" {
		t.Fatalf("want error field; body=%s", resp.Body)
	}
	low := strings.ToLower(resp.Error + " " + resp.Body)
	if !strings.Contains(low, "dir") && !strings.Contains(low, "directory") && !strings.Contains(low, "not a") {
		t.Logf("error text lacks dir wording (ok if clear): %q", resp.Error)
	}
}
```
