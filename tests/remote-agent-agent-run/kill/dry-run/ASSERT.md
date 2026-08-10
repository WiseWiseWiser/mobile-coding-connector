## Expected

1. Exit 0.
2. Output mentions dry-run / would stop / sess-k (or KillDryRun true when inject wired).
3. Soft: KillCalled && KillDryRun.

## Exit Code

0.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("kill dry-run failed: %s", resp.Combined)
	}
	if resp.KillCalled && !resp.KillDryRun {
		t.Fatalf("expected KillDryRun=true when inject observed")
	}
	msg := strings.ToLower(resp.Stdout + "\n" + resp.Combined)
	if resp.KillCalled {
		return
	}
	// Fallback text signals
	if !strings.Contains(msg, "dry") && !strings.Contains(msg, "would") &&
		!strings.Contains(msg, "sess-k") {
		t.Fatalf("expected dry-run report; combined:\n%s", resp.Combined)
	}
}
```
