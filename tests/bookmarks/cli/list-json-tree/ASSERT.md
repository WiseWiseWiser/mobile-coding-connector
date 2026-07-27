## Expected

1. Exit 0.
2. Stdout parses as JSON with version and roots; includes JSON Item or bm_j.

## Errors

- Non-JSON; missing keys.

```go
import (
	"encoding/json"
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q", resp.ExitCode, resp.Combined)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp.Stdout)), &raw); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, resp.Stdout)
	}
	if _, ok := raw["version"]; !ok {
		t.Fatalf("missing version: %v", raw)
	}
	if _, ok := raw["roots"]; !ok {
		t.Fatalf("missing roots: %v", raw)
	}
	s := resp.Stdout
	if !strings.Contains(s, "JSON Item") && !strings.Contains(s, "bm_j") {
		t.Fatalf("seed item missing from json: %s", s)
	}
}
```
