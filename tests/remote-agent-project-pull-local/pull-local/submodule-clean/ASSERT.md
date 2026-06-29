## Expected

1. Exit code 0.
2. Worktree created; remote clean after pull.

## Side Effects

Standard successful pull with submodule guard passing.

## Errors

- Non-zero exit.

## Exit Code

0.

```go
import (
	"os"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	base := worktreeBaseDir(resp.AgentHome)
	if _, err := os.Stat(base); err != nil {
		t.Fatalf("worktree base: %v", err)
	}
	if strings.TrimSpace(gitPorcelain(t, resp.ProjectDir)) != "" {
		t.Fatalf("remote should be clean after pull")
	}
}
```