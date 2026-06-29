## Expected

1. Exit code 0.
2. Stdout contains project header `Project: clean-repo-test (clean-001)`.
3. Stdout contains `Git Branch:       main`.
4. Stdout matches `Git Commit:` with a 7-char hex hash followed by two spaces and `Initial commit`.
5. Stdout contains `Worktree:         clean`.

## Side Effects

None beyond subprocess startup and temp dir cleanup.

## Errors

- Missing git status lines or wrong worktree state.

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
	checks := []string{
		"Project: clean-repo-test (clean-001)",
		"Git Branch:       main",
		"Worktree:         clean",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q;\n%s", want, out)
		}
	}

	commitRe := regexp.MustCompile(`Git Commit:\s+[0-9a-f]{7}  Initial commit`)
	if !commitRe.MatchString(out) {
		t.Fatalf("stdout missing Git Commit line with 7-char hash and message;\n%s", out)
	}
}
```