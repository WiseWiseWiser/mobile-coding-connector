## Expected

- Check `gateway_running` status `warn`.
- Hint mentions `openclaw start`.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	check := doctorCheck(resp.Doctor, "gateway_running")
	if check == nil || check.Status != openclaw.DoctorWarn {
		t.Fatalf("gateway_running = %+v", check)
	}
	if !strings.Contains(check.Hint, "openclaw start") {
		t.Fatalf("hint = %q, want start command", check.Hint)
	}
}
```