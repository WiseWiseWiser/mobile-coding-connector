## Expected

1. Harness `err` nil.
2. `DoctorErr` empty (all checks pass).
3. `Doctor.PairName` is `mad-max`.
4. `Doctor.AllOK` true.
5. Each check name present and OK: `pairs-json`, `local-version`, `remote-version`,
   `versions-match`, `serve`, `local-root`, `remote-root`, `profile`.

## Side Effects

- None required beyond seeds.

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
		t.Fatalf("Doctor error: %s (report=%+v)", resp.DoctorErr, resp.Doctor)
	}
	if resp.Doctor.PairName != "mad-max" {
		t.Fatalf("PairName: got %q want mad-max", resp.Doctor.PairName)
	}
	if !resp.Doctor.AllOK {
		t.Fatalf("AllOK false; checks=%+v", resp.Doctor.Checks)
	}
	for _, name := range []string{
		"pairs-json", "local-version", "remote-version", "versions-match",
		"serve", "local-root", "remote-root", "profile",
	} {
		c := checkNamed(resp.Doctor, name)
		if c == nil {
			t.Fatalf("missing check %q; checks=%+v", name, resp.Doctor.Checks)
		}
		if !c.OK {
			t.Fatalf("check %q not OK: detail=%q", name, c.Detail)
		}
	}
}
```
