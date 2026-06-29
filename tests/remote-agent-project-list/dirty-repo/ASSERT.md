## Expected

1. Exit code 0.
2. Stdout contains project header `Project: dirty-repo-test (dirty-001)`.
3. Stdout contains `Git Branch:       main`.
4. Stdout contains a `Git Commit:` line with 7-char hash and `Initial commit`.
5. Stdout contains exactly:
   `Worktree:         dirty (1 added, 1 changed, 1 renamed, 1 deleted)`.

## Setup-derived counts

| Change | File | Porcelain type |
|--------|------|----------------|
| Added | `untracked.txt` (untracked) | added |
| Changed | `tracked.txt` (modified) | changed |
| Renamed | `to-rename.txt` → `renamed.txt` | renamed |
| Deleted | `to-delete.txt` (removed) | deleted |

## Side Effects

None beyond subprocess startup and temp dir cleanup.

## Errors

- Wrong dirty counts or `Worktree: clean`.

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
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	out := resp.Stdout
	if !strings.Contains(out, "Project: dirty-repo-test (dirty-001)") {
		t.Fatalf("missing project header;\n%s", out)
	}
	if !strings.Contains(out, "Git Branch:       main") {
		t.Fatalf("missing branch line;\n%s", out)
	}

	commitRe := regexp.MustCompile(`Git Commit:\s+[0-9a-f]{7}  Initial commit`)
	if !commitRe.MatchString(out) {
		t.Fatalf("missing Git Commit line;\n%s", out)
	}

	wantWorktree := "Worktree:         dirty (1 added, 1 changed, 1 renamed, 1 deleted)"
	if !strings.Contains(out, wantWorktree) {
		t.Fatalf("stdout missing %q;\n%s", wantWorktree, out)
	}
}
```