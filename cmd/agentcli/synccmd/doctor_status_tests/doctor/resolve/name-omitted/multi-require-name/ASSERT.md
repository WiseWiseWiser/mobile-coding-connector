## Expected

1. Harness err nil.
2. `DoctorErr` contains `pair name required`.
3. AllOK not true.

## Side Effects

- None.

## Errors

- Substring: `pair name required`.

## Exit Code

- Non-nil library error.

```go
import (
	"strings"
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
	if resp.DoctorErr == "" {
		t.Fatal("expected pair name required error")
	}
	if !strings.Contains(resp.DoctorErr, "pair name required") {
		t.Fatalf("want pair name required in error; got %q", resp.DoctorErr)
	}
	if resp.Doctor.AllOK {
		t.Fatal("AllOK must not be true")
	}
}
```
