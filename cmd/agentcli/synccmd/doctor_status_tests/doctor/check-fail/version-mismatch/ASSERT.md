## Expected

1. Harness err nil.
2. `DoctorErr` non-empty (critical check failed).
3. `Doctor.AllOK` false.
4. Check `versions-match` present with OK=false.
5. Local and remote version checks may still be OK individually.

## Side Effects

- None.

## Errors

- Non-nil Doctor error.

## Exit Code

- Non-nil library error.

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
	if resp.DoctorErr == "" {
		t.Fatal("expected Doctor error on version mismatch")
	}
	if resp.Doctor.AllOK {
		t.Fatal("AllOK true; want false on version mismatch")
	}
	c := checkNamed(resp.Doctor, "versions-match")
	if c == nil {
		t.Fatalf("missing versions-match check; checks=%+v", resp.Doctor.Checks)
	}
	if c.OK {
		t.Fatalf("versions-match should fail; detail=%q", c.Detail)
	}
}
```
