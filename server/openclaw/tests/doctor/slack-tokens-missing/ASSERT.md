## Expected

- `slack_tokens` status `fail` with non-empty hint.
- `Doctor.Healthy` is false.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	check := doctorCheck(resp.Doctor, "slack_tokens")
	if check == nil || check.Status != openclaw.DoctorFail {
		t.Fatalf("slack_tokens = %+v", check)
	}
	if check.Hint == "" {
		t.Fatal("slack_tokens fail should include hint")
	}
	if resp.Doctor.Healthy {
		t.Fatal("doctor should be unhealthy when slack tokens missing")
	}
}
```