## Expected

- Exit code 0.
- Output contains `mode=git-fetch` and remote dir.
- Remote still dirty (dry-run).

## Exit Code

0.

```go
import (
	"os/exec"
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
	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "git-fetch") {
		t.Fatalf("expected git-fetch in plan;\n%s", resp.Combined)
	}
	if !strings.Contains(combined, "dry-run") {
		t.Fatalf("expected dry-run banner;\n%s", resp.Combined)
	}
	remote := req.RemoteDirAfterSetup
	if remote == "" {
		t.Fatal("missing RemoteDirAfterSetup")
	}
	out, err := exec.Command("git", "-C", remote, "status", "--porcelain").CombinedOutput()
	if err != nil {
		t.Fatalf("git status: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) == "" {
		t.Fatal("remote should remain dirty after dry-run")
	}
}
```
