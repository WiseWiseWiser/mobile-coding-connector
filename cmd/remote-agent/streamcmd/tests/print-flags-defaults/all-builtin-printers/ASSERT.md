## Expected

1. `RunErr` is empty.
2. Stdout contains `  hello log` (log default indent).
3. Stdout contains `Server checks:` (section title + colon).
4. Stdout contains `[ok] configuration load:` (progress check formatting).
5. Stdout contains `/data/ws-proxy.json` (progress detail).
6. Stderr is empty (no B-path override).

## Side Effects

None.

## Errors

- Missing builtin formatting when Print flags set.
- Output buffered only at end (all lines should be present in captured buffer).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("streamcmd.Run error: %s", resp.RunErr)
	}
	if !strings.Contains(resp.Stdout, "  hello log") {
		t.Fatalf("stdout missing log line; got:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "Server checks:") {
		t.Fatalf("stdout missing section header; got:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "[ok] configuration load:") {
		t.Fatalf("stdout missing progress check; got:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "/data/ws-proxy.json") {
		t.Fatalf("stdout missing progress detail; got:\n%s", resp.Stdout)
	}
	if resp.Stderr != "" {
		t.Fatalf("stderr should be empty, got:\n%s", resp.Stderr)
	}
}
```
