## Expected

1. HTTP status `200`.
2. Body has `ok: true` (`resp.OK`).
3. `OpenCalled` is true; `RecordedDir` equals request dir (or resolved abs path containing it).
4. `RecordedMode` is `ModeReuseCurrent` (not ModeSmart zero-value).
5. When script recorded: contains path and reuse markers (`matchingSession` or `matchingTab`).

## Errors

- Defaulting to ModeSmart (Config zero value).
- Success body missing `{"ok":true}`.

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
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, resp.Body)
	}
	if !resp.OK {
		t.Fatalf("want ok:true in body; body=%s", resp.Body)
	}
	if resp.Error != "" {
		t.Fatalf("unexpected error field: %q body=%s", resp.Error, resp.Body)
	}
	if !resp.OpenCalled {
		t.Fatal("Open was not called")
	}
	if resp.RecordedMode != iterm2.ModeReuseCurrent {
		t.Fatalf("RecordedMode = %v, want ModeReuseCurrent", resp.RecordedMode)
	}
	if !strings.Contains(resp.RecordedDir, req.Dir) && resp.RecordedDir != req.Dir {
		if !strings.HasSuffix(resp.RecordedDir, filepathBase(req.Dir)) {
			t.Fatalf("RecordedDir = %q, want related to %q", resp.RecordedDir, req.Dir)
		}
	}
	if resp.RecordedScript != "" {
		if !strings.Contains(resp.RecordedScript, "matchingSession") &&
			!strings.Contains(resp.RecordedScript, "matchingTab") {
			t.Fatalf("reuse script missing session markers: %q", resp.RecordedScript)
		}
	}
}

func filepathBase(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return p
	}
	return p[i+1:]
}
```
