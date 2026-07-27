## Expected

1. Exit 0.
2. Stdout is a JSON array including port 3000.
3. No ANSI color codes in stdout.

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
	if strings.Contains(resp.Stdout, "\x1b[") || strings.Contains(resp.Stdout, "\033") {
		t.Fatalf("--json must not use ANSI; stdout:\n%s", resp.Stdout)
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &rows); err != nil {
		t.Fatalf("stdout not JSON array: %v\n%s", err, resp.Stdout)
	}
	found := false
	for _, r := range rows {
		// accept port or localPort keys
		for _, k := range []string{"port", "localPort", "Port"} {
			if v, ok := r[k]; ok {
				switch n := v.(type) {
				case float64:
					if int(n) == 3000 {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Fatalf("JSON missing port 3000; stdout:\n%s", resp.Stdout)
	}
}
```
