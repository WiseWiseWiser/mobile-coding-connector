## Expected

1. Harness `err` is nil.
2. `CreateErr` is empty.
3. `SessionID` is non-empty.

## Side Effects

- Manager holds a live session until server close (httptest cleanup).

## Errors

None expected on happy path.

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
	if resp.CreateErr != "" {
		t.Fatalf("CreateSSHSession error: %s", resp.CreateErr)
	}
	if resp.SessionID == "" {
		t.Fatal("SessionID must be non-empty after authorized create")
	}
}
```
