## Expected

- `slack_tokens` status `ok`.
- `slack_plugin` and `slack_socket` status `warn`.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	tokens := doctorCheck(resp.Doctor, "slack_tokens")
	if tokens == nil || tokens.Status != openclaw.DoctorOK {
		t.Fatalf("slack_tokens = %+v", tokens)
	}
	plugin := doctorCheck(resp.Doctor, "slack_plugin")
	if plugin == nil || plugin.Status != openclaw.DoctorWarn {
		t.Fatalf("slack_plugin = %+v", plugin)
	}
	socket := doctorCheck(resp.Doctor, "slack_socket")
	if socket == nil || socket.Status != openclaw.DoctorWarn {
		t.Fatalf("slack_socket = %+v", socket)
	}
}
```