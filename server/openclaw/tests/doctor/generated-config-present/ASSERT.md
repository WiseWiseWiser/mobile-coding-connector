## Expected

- Check `generated_config` status `ok`.
- Detail points at generated file path under data dir.

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
	check := doctorCheck(resp.Doctor, "generated_config")
	if check == nil || check.Status != openclaw.DoctorOK {
		t.Fatalf("generated_config = %+v", check)
	}
	if !strings.Contains(check.Detail, "openclaw.json") {
		t.Fatalf("detail = %q, want generated path", check.Detail)
	}
}
```