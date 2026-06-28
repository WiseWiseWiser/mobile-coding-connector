## Expected

- Check `gateway_running` status `ok`.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	check := doctorCheck(resp.Doctor, "gateway_running")
	if check == nil || check.Status != openclaw.DoctorOK {
		t.Fatalf("gateway_running = %+v", check)
	}
}
```