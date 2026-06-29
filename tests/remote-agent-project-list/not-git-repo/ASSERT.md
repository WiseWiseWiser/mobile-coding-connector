## Expected

1. Exit code 0.
2. Stdout contains `Project: not-git-test (not-git-001)`.
3. Stdout contains `Dir:` with the registered absolute path.
4. Stdout contains:
   - `Git Branch:       -`
   - `Git Commit:       -`
   - `Worktree:         -`

## Side Effects

None beyond subprocess startup and temp dir cleanup.

## Errors

- Any real branch name or commit hash shown for a non-git dir.

## Exit Code

0.

```go
import (
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

	out := resp.Stdout
	if !strings.Contains(out, "Project: not-git-test (not-git-001)") {
		t.Fatalf("missing project header;\n%s", out)
	}
	if !strings.Contains(out, resp.ProjectDir) {
		t.Fatalf("stdout missing registered dir %q;\n%s", resp.ProjectDir, out)
	}

	dashLines := []string{
		"Git Branch:       -",
		"Git Commit:       -",
		"Worktree:         -",
	}
	for _, want := range dashLines {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q;\n%s", want, out)
		}
	}
}
```