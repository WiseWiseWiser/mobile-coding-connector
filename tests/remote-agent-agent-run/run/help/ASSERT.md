## Expected

1. Exit 0.
2. Mentions `run` and at least two of: session, detach, open, dir, prompt, json, auto-send.
3. Soft: may mention --new-terminal as rejected/unavailable.

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
		t.Fatalf("exit %d; %s", resp.ExitCode, resp.Combined)
	}
	out := strings.ToLower(resp.Stdout)
	if !strings.Contains(out, "run") {
		t.Fatalf("help must mention run; stdout:\n%s", resp.Stdout)
	}
	n := 0
	for _, k := range []string{"session", "detach", "open", "dir", "prompt", "json", "auto-send", "agent-runner"} {
		if strings.Contains(out, k) {
			n++
		}
	}
	if n < 2 {
		t.Fatalf("run help should document core flags; stdout:\n%s", resp.Stdout)
	}
}
```
