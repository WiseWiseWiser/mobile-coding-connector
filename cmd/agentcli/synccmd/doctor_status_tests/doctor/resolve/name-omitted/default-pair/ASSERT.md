## Expected

1. Harness err nil.
2. `DoctorErr` empty (happy path after resolve).
3. `Doctor.PairName` is `alpha` (defaultPair), not beta.
4. `Doctor.AllOK` true.

## Side Effects

- None beyond seeds.

## Errors

- None.

## Exit Code

- Nil library error.

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
	if resp.DoctorErr != "" {
		t.Fatalf("Doctor error: %s", resp.DoctorErr)
	}
	if resp.Doctor.PairName != "alpha" {
		t.Fatalf("PairName: got %q want alpha (defaultPair)", resp.Doctor.PairName)
	}
	if !resp.Doctor.AllOK {
		t.Fatalf("AllOK false; checks=%+v", resp.Doctor.Checks)
	}
}
```
