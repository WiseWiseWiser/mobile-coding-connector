## Expected

1. Harness err nil.
2. `DoctorErr` non-empty.
3. `Doctor.AllOK` false.
4. Check `serve` present with OK=false; detail may mention connection/serve.

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
		t.Fatal("expected Doctor error when serve is down")
	}
	if resp.Doctor.AllOK {
		t.Fatal("AllOK true; want false when serve down")
	}
	c := checkNamed(resp.Doctor, "serve")
	if c == nil {
		t.Fatalf("missing serve check; checks=%+v", resp.Doctor.Checks)
	}
	if c.OK {
		t.Fatalf("serve should fail; detail=%q", c.Detail)
	}
}
```
