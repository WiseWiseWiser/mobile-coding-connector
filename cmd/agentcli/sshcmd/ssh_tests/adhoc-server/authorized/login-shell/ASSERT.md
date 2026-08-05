## Expected

1. Harness `err` is nil.
2. `AuthErr` empty.
3. `ShellOK` is true (stdout contained `shell-ok` after Shell).
4. `ShellErr` empty.
5. `AdhocPort` > 0.

## Side Effects

- AdhocServer closed after Run.

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
	if resp.AuthErr != "" {
		t.Fatalf("SSH auth error: %s", resp.AuthErr)
	}
	if resp.ShellErr != "" {
		t.Fatalf("login shell error: %s (stdout=%q)", resp.ShellErr, resp.Stdout)
	}
	if !resp.ShellOK {
		t.Fatalf("ShellOK false; stdout=%q", resp.Stdout)
	}
	if resp.AdhocPort <= 0 {
		t.Fatalf("AdhocPort: got %d want > 0", resp.AdhocPort)
	}
}
```
