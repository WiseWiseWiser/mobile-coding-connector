## Expected

1. Non-zero exit.
2. Stderr (or combined) contains `ai-critic` (start hint).
3. Output references `http://localhost:23712` or the attempted URL.

## Side Effects

No server subprocess.

## Errors

- Missing start hint when server is down.
- Hint appears on unrelated failures (covered in sibling leaf).

## Exit Code

Non-zero.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure when server down; combined:\n%s", resp.Combined)
	}
	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "ai-critic") {
		t.Fatalf("expected ai-critic start hint; combined:\n%s", resp.Combined)
	}
}
```