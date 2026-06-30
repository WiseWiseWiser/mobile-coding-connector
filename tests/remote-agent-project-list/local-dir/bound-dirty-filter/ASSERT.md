## Expected

1. Exit code 0.
2. Dirty project listed with bound `Local Dir` path and dirty worktree line.
3. No `Local Dir` dash-only regression for this project.

## Side Effects

None.

## Errors

- Missing Local Dir on dirty filtered project.

## Exit Code

0.

```go
import (
	"fmt"
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
	if resp.LocalPath == "" {
		t.Fatal("LocalPath not set")
	}
	wantLocal := fmt.Sprintf("Local Dir:        %s", resp.LocalPath)
	out := resp.Stdout
	if !strings.Contains(out, "Project: local-dirty-bound (local-dirty-001)") {
		t.Fatalf("missing project;\n%s", out)
	}
	if !strings.Contains(out, wantLocal) {
		t.Fatalf("missing bound local dir %q;\n%s", wantLocal, out)
	}
	if !strings.Contains(out, "Worktree:         dirty") {
		t.Fatalf("missing dirty worktree;\n%s", out)
	}
}
```