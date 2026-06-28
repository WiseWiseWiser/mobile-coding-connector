## Expected

- Checks `node` and `openclaw_cli` exist.
- Status is `ok` or `fail` (never `skip`).
- On `fail`, `hint` is non-empty.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	for _, id := range []string{"node", "openclaw_cli"} {
		check := doctorCheck(resp.Doctor, id)
		if check == nil {
			t.Fatalf("missing check %q", id)
		}
		if check.Status != openclaw.DoctorOK && check.Status != openclaw.DoctorFail {
			t.Fatalf("%s status = %q, want ok or fail", id, check.Status)
		}
		if check.Status == openclaw.DoctorFail && check.Hint == "" {
			t.Fatalf("%s fail should include hint", id)
		}
	}
}
```