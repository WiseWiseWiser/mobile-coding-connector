## Expected

1. Status 200 with `ok:true`; `RecordedMode` is `ModeForceNew`.
2. Script (when present) has `create window` and does not scan `matchingSession`.

```go
import (
	"strings"
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
	if resp.RecordedMode != iterm2.ModeForceNew {
		t.Fatalf("mode = %v, want ModeForceNew", resp.RecordedMode)
	}
	if resp.RecordedScript != "" {
		if !strings.Contains(resp.RecordedScript, "create window with default profile") {
			t.Fatalf("force-new script missing create window: %q", resp.RecordedScript)
		}
		if strings.Contains(resp.RecordedScript, "matchingSession") {
			t.Fatalf("force-new script should not scan matchingSession: %q", resp.RecordedScript)
		}
	}
}
```
