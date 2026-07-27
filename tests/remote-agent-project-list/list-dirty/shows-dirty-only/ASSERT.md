## Expected

- Exit code 0
- Stdout contains `dirty-project` and `Worktree:         dirty`
- Stdout does not contain `clean-project`

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
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	if !strings.Contains(resp.Stdout, "Project: dirty-project (dirty-001)") {
		t.Fatalf("stdout missing dirty project:\n%s", resp.Stdout)
	}
	if !strings.Contains(resp.Stdout, "Worktree:         dirty") {
		t.Fatalf("stdout missing dirty worktree line:\n%s", resp.Stdout)
	}
	if strings.Contains(resp.Stdout, "clean-project") {
		t.Fatalf("stdout must not contain clean project:\n%s", resp.Stdout)
	}
}
```