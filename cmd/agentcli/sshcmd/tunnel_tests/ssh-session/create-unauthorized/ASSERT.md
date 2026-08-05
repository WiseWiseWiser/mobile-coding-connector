## Expected

1. Harness `err` is nil (failure is in Response, not harness).
2. `CreateErr` is non-empty.
3. `SessionID` is empty.

## Side Effects

- No authorized session usable by this client.

## Errors

CreateSession must surface HTTP auth failure (401/403 or decodeable API error).

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.CreateErr == "" {
		t.Fatal("CreateSSHSession must fail with wrong token (CreateErr empty)")
	}
	if resp.SessionID != "" {
		t.Fatalf("SessionID must be empty on unauthorized create; got %q", resp.SessionID)
	}
}
```
