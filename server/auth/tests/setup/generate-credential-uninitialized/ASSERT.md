## Expected

On an uninitialized server, `POST /api/auth/credentials/generate` must succeed:

1. `Response.StatusCode` is `200`.
2. `Response.JSON["credential"]` is a non-empty string.
3. The credential matches `^[0-9a-f]{64}$` (32 random bytes SHA-256 hex).
4. `Response.JSON["error"]` is absent.

## Errors

- Status 401 with `{"error":"not_initialized"}` (current bug).
- Missing or malformed credential field.

```go
import (
	"regexp"
	"testing"
)

var hexCredentialPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("POST /api/auth/credentials/generate: status=%d body=%s; want 200 with credential (got not_initialized when endpoint is blocked by middleware)", resp.StatusCode, resp.Body)
	}

	if resp.JSON == nil {
		t.Fatalf("expected JSON body, got: %s", resp.Body)
	}

	if errVal, ok := resp.JSON["error"]; ok {
		t.Fatalf("unexpected error field %v in body: %s", errVal, resp.Body)
	}

	cred, _ := resp.JSON["credential"].(string)
	if cred == "" {
		t.Fatalf("credential missing or empty in body: %s", resp.Body)
	}
	if !hexCredentialPattern.MatchString(cred) {
		t.Fatalf("credential %q does not match 64-char hex pattern", cred)
	}

	t.Logf("generated credential length=%d", len(cred))
}
```