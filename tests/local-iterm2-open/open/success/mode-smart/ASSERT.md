## Expected

1. Status 200 with `ok:true`; `RecordedMode` is `ModeSmart`.
2. Script (when present) includes smart-open tab/window branches.

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
	if resp.RecordedMode != iterm2.ModeSmart {
		t.Fatalf("mode = %v, want ModeSmart", resp.RecordedMode)
	}
	if resp.RecordedScript != "" {
		if !strings.Contains(resp.RecordedScript, "create tab") {
			t.Fatalf("smart script missing create tab: %q", resp.RecordedScript)
		}
		if !strings.Contains(resp.RecordedScript, "create window with default profile") {
			t.Fatalf("smart script missing create window fallback: %q", resp.RecordedScript)
		}
	}
}
```
