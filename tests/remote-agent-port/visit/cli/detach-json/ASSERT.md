## Expected

1. Exit 0.
2. Stdout is JSON with public_url (or publicUrl), provider, port/id, idle_seconds, status.
3. No ANSI in stdout.
4. Sessions still non-empty on server after CLI returns.

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
	if strings.Contains(resp.Stdout, "\x1b[") {
		t.Fatalf("--json must not use ANSI; stdout:\n%s", resp.Stdout)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &m); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, resp.Stdout)
	}
	url, _ := m["public_url"].(string)
	if url == "" {
		url, _ = m["publicUrl"].(string)
	}
	if url == "" {
		t.Fatalf("JSON missing public_url: %s", resp.Stdout)
	}
	if len(resp.Sessions) == 0 {
		t.Fatal("session must remain on server after --detach")
	}
}
```
