## Expected

1. Exit code 0.
2. Stdout contains `Project: detached-head-test (detached-001)`.
3. Stdout contains `Git Branch:       (detached)`.
4. Stdout matches `Git Commit:` with 7-char hash and `Initial commit`.
5. Stdout contains `Worktree:         clean`.

## Side Effects

None beyond subprocess startup and temp dir cleanup.

## Errors

- Branch shows `main` instead of `(detached)`.
- Missing commit hash or message.

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
		"Project: detached-head-test (detached-001)",
		"Git Branch:       (detached)",
		"Worktree:         clean",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q;\n%s", want, out)
		}
	}

	commitRe := regexp.MustCompile(`Git Commit:\s+[0-9a-f]{7}  Initial commit`)
	if !commitRe.MatchString(out) {
		t.Fatalf("stdout missing Git Commit line;\n%s", out)
	}
}
```