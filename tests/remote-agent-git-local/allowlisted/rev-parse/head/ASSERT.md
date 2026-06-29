## Expected

1. Exit 0.
2. Stdout is a full 40-character lowercase hex commit hash (optional single trailing newline).

## Exit Code

0.

```go
import (
	"regexp"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; stdout:\n%s", resp.ExitCode, resp.Stdout)
	}
	hash := strings.TrimSpace(resp.Stdout)
	re := regexp.MustCompile(`^[0-9a-f]{40}$`)
	if !re.MatchString(hash) {
		t.Fatalf("stdout should be 40-char hash, got %q", hash)
	}
}
```