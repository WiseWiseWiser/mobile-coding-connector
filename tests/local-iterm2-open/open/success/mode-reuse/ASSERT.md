## Expected

1. Status 200 with `ok:true`.
2. Open called with `ModeReuseCurrent`.

```go
import (
	"testing"

	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d body=%s", resp.StatusCode, resp.Body)
	}
	if !resp.OK {
		t.Fatalf("want ok:true; body=%s", resp.Body)
	}
	if !resp.OpenCalled {
		t.Fatal("Open not called")
	}
	if resp.RecordedMode != iterm2.ModeReuseCurrent {
		t.Fatalf("mode = %v, want ModeReuseCurrent", resp.RecordedMode)
	}
}
```
