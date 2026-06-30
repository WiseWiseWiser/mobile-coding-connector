## Expected

1. Exit code 0.
2. Stdout contains project header `Project: identity-test (identity-001)`.
3. New git status lines present:
   - `Git Branch:       main`
   - `Git Commit:` with 7-char hash and `Initial commit`
   - `Worktree:         clean`
4. Identity lines unchanged:
   - `Git Identity ID:  mp663i1zlyx3th`
   - `Git User Name:    xhd2015`
   - `Git User Email:   xhd2015@gmail.com`
5. Git status lines appear **before** identity lines (branch line precedes identity ID line in stdout).

## Side Effects

None beyond subprocess startup and temp dir cleanup.

## Errors

- Missing identity fields or missing new git status lines.

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

	gitLines := []string{
		"Git Branch:       main",
		"Worktree:         clean",
	}
	for _, want := range gitLines {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q;\n%s", want, out)
		}
	}

	commitRe := regexp.MustCompile(`Git Commit:\s+[0-9a-f]{7}  Initial commit`)
	if !commitRe.MatchString(out) {
		t.Fatalf("stdout missing Git Commit line;\n%s", out)
	}

	identityLines := []string{
		"Project: identity-test (identity-001)",
		"Git Identity ID:  mp663i1zlyx3th",
		"Git User Name:    xhd2015",
		"Git User Email:   xhd2015@gmail.com",
	}
	for _, want := range identityLines {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q;\n%s", want, out)
		}
	}

	if !strings.Contains(out, localDirDashLine) {
		t.Fatalf("stdout missing %q;\n%s", localDirDashLine, out)
	}
	dirIdx := strings.Index(out, "  Dir:")
	localIdx := strings.Index(out, "Local Dir:")
	branchIdx := strings.Index(out, "Git Branch:")
	identityIdx := strings.Index(out, "Git Identity ID:")
	if dirIdx < 0 || localIdx < 0 || branchIdx < 0 || identityIdx < 0 {
		t.Fatalf("could not locate field order markers;\n%s", out)
	}
	if !(dirIdx < localIdx && localIdx < branchIdx && branchIdx < identityIdx) {
		t.Fatalf("expected Dir < Local Dir < Git Branch < identity lines;\n%s", out)
	}
}
```