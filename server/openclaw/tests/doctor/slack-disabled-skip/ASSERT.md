## Expected

- Check `slack_enabled` has status `skip`.
- No `slack_tokens`, `slack_plugin`, or `slack_socket` checks.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	check := doctorCheck(resp.Doctor, "slack_enabled")
	if check == nil || check.Status != openclaw.DoctorSkip {
		t.Fatalf("slack_enabled = %+v", check)
	}
	for _, id := range []string{"slack_tokens", "slack_plugin", "slack_socket"} {
		if doctorCheck(resp.Doctor, id) != nil {
			t.Fatalf("unexpected check %q when slack disabled", id)
		}
	}
}
```