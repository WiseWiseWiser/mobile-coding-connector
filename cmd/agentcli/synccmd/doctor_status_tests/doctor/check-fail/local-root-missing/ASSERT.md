## Expected

1. Harness err nil.
2. `DoctorErr` non-empty.
3. `Doctor.AllOK` false.
4. Check `local-root` present with OK=false.

## Side Effects

- LocalPath removed by harness before Doctor.

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
		t.Fatal("expected Doctor error when local root missing")
	}
	if resp.Doctor.AllOK {
		t.Fatal("AllOK true; want false when local root missing")
	}
	c := checkNamed(resp.Doctor, "local-root")
	if c == nil {
		t.Fatalf("missing local-root check; checks=%+v", resp.Doctor.Checks)
	}
	if c.OK {
		t.Fatalf("local-root should fail; detail=%q", c.Detail)
	}
}
```
