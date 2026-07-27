## Expected

1. Exit 0.
2. Stderr contains `warning:` (not-listening).
3. Session still created (JSON on stdout or Sessions non-empty).

## Exit Code

0.

```go
import (
	"encoding/json"
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(strings.ToLower(resp.Stderr), "warning") {
		t.Fatalf("expected warning: on stderr; stderr:\n%s", resp.Stderr)
	}
	// Session started despite warn
	if len(resp.Sessions) == 0 {
		// try parse stdout JSON
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &m); err != nil || m["public_url"] == nil && m["publicUrl"] == nil {
			t.Fatalf("expected active session after warn; sessions=%v stdout=%s", resp.Sessions, resp.Stdout)
		}
	}
}
```
