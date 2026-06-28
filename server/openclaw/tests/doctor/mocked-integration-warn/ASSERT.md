## Expected

- `Doctor.Mocked` is true.
- Check `mock_mode` has status `warn`.

```go
import (
	"testing"

	"github.com/xhd2015/ai-critic/server/openclaw"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !resp.Doctor.Mocked {
		t.Fatal("doctor.mocked should be true")
	}
	check := doctorCheck(resp.Doctor, "mock_mode")
	if check == nil || check.Status != openclaw.DoctorWarn {
		t.Fatalf("mock_mode check = %+v", check)
	}
}
```