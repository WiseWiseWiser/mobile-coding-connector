## Expected

1. Exit 0.
2. OpenDryRun/Combined contains `firefox` and `https://open.example.com`.
3. Does not require real browser process.

## Errors

- Non-zero; missing browser/url in output.

```go
import (
	"strings"
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit=%d out=%q err=%q", resp.ExitCode, resp.Combined, resp.ErrMsg)
	}
	out := strings.ToLower(resp.OpenDryRun)
	if out == "" {
		out = strings.ToLower(resp.Combined)
	}
	if !strings.Contains(out, "firefox") {
		t.Fatalf("expected effective browser firefox in output: %q", resp.Combined)
	}
	if !strings.Contains(out, "https://open.example.com") {
		t.Fatalf("expected url in output: %q", resp.Combined)
	}
}
```
